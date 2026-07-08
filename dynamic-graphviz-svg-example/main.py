import json
import subprocess
import os
from typing import Dict, Any, Optional
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
from pydantic import BaseModel
import pydot

app = FastAPI(title="Dynamic Graphviz SVG Studio")

# CORS middleware for local development
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Models for Request Payloads
class RenderRequest(BaseModel):
    dot_code: str
    engine: str = "dot"

class UpdateRequest(BaseModel):
    dot_code: str
    action: str  # add_node, add_edge, delete_node, delete_edge, update_node, update_edge, update_graph
    engine: str = "dot"
    params: Dict[str, Any]

# Helper to run graphviz commands
def render_graphviz(dot_code: str, engine: str = "dot") -> Dict[str, Any]:
    valid_engines = {"dot", "neato", "fdp", "sfdp", "twopi", "circo"}
    if engine not in valid_engines:
        return {"success": False, "error": f"Invalid layout engine: {engine}. Choose from {valid_engines}"}
    
    # Render SVG
    try:
        process_svg = subprocess.run(
            ["dot", f"-K{engine}", "-Tsvg"],
            input=dot_code,
            capture_output=True,
            text=True,
            check=True
        )
        svg_output = process_svg.stdout
    except subprocess.CalledProcessError as e:
        error_msg = e.stderr or e.stdout or str(e)
        return {"success": False, "error": f"Graphviz render error:\n{error_msg}"}
    
    # Render JSON layout representation
    try:
        process_json = subprocess.run(
            ["dot", f"-K{engine}", "-Tjson"],
            input=dot_code,
            capture_output=True,
            text=True,
            check=True
        )
        json_output = json.loads(process_json.stdout)
    except subprocess.CalledProcessError as e:
        json_output = None
    except Exception:
        json_output = None
        
    return {
        "success": True,
        "svg": svg_output,
        "graph_data": json_output
    }

# Endpoints
@app.post("/api/render")
def api_render(req: RenderRequest):
    res = render_graphviz(req.dot_code, req.engine)
    if not res["success"]:
        return {"success": False, "error": res["error"]}
    return res

@app.post("/api/update")
def api_update(req: UpdateRequest):
    # Parse DOT
    try:
        graphs = pydot.graph_from_dot_data(req.dot_code)
        if not graphs:
            return {"success": False, "error": "Invalid DOT code: No graph found."}
        graph = graphs[0]
    except Exception as e:
        return {"success": False, "error": f"Failed to parse DOT: {str(e)}"}

    action = req.action
    params = req.params

    try:
        if action == "add_node":
            node_id = params.get("node_id")
            attrs = params.get("attributes", {})
            if not node_id:
                return {"success": False, "error": "Missing 'node_id' parameter"}
            
            # Clean empty/none attributes
            attrs = {k: str(v) for k, v in attrs.items() if v is not None and v != ""}
            
            # Create/update node
            nodes = graph.get_node(node_id)
            if nodes:
                node = nodes[0]
                for k, v in attrs.items():
                    node.set(k, v)
            else:
                node = pydot.Node(node_id, **attrs)
                graph.add_node(node)
                
        elif action == "add_edge":
            source = params.get("source")
            target = params.get("target")
            attrs = params.get("attributes", {})
            if not source or not target:
                return {"success": False, "error": "Missing 'source' or 'target' parameters"}
            
            attrs = {k: str(v) for k, v in attrs.items() if v is not None and v != ""}
            
            edge = pydot.Edge(source, target, **attrs)
            graph.add_edge(edge)
            
        elif action == "delete_node":
            node_id = params.get("node_id")
            if not node_id:
                return {"success": False, "error": "Missing 'node_id' parameter"}
            
            # Delete explicit node definitions
            nodes = graph.get_node(node_id)
            for n in nodes:
                graph.del_node(n)
                
            # pydot del_node only deletes the node object. We also need to search and delete edges.
            edges_to_delete = []
            for e in graph.get_edges():
                src = e.get_source().strip('"')
                dst = e.get_destination().strip('"')
                target_id = node_id.strip('"')
                if src == target_id or dst == target_id:
                    edges_to_delete.append(e)
            
            for e in edges_to_delete:
                graph.del_edge(e.get_source(), e.get_destination())

            # Also verify if the node was added as a plain identifier and delete it
            # (sometimes pydot keeps it in the graph elements structure)
            graph.del_node(node_id)

        elif action == "delete_edge":
            source = params.get("source")
            target = params.get("target")
            if not source or not target:
                return {"success": False, "error": "Missing 'source' or 'target' parameters"}
            
            # Delete edge from pydot
            graph.del_edge(source, target)
            
        elif action == "update_node":
            node_id = params.get("node_id")
            attrs = params.get("attributes", {})
            if not node_id:
                return {"success": False, "error": "Missing 'node_id' parameter"}
            
            nodes = graph.get_node(node_id)
            if nodes:
                node = nodes[0]
            else:
                node = pydot.Node(node_id)
                graph.add_node(node)
                
            for k, v in attrs.items():
                if v is None or v == "":
                    # Delete attribute if empty
                    if k in node.get_attributes():
                        del node.get_attributes()[k]
                else:
                    node.set(k, str(v))
                    
        elif action == "update_edge":
            source = params.get("source")
            target = params.get("target")
            attrs = params.get("attributes", {})
            if not source or not target:
                return {"success": False, "error": "Missing 'source' or 'target' parameters"}
            
            edges = graph.get_edge(source, target)
            if edges:
                edge = edges[0]
            else:
                edge = pydot.Edge(source, target)
                graph.add_edge(edge)
                
            for k, v in attrs.items():
                if v is None or v == "":
                    if k in edge.get_attributes():
                        del edge.get_attributes()[k]
                else:
                    edge.set(k, str(v))
                    
        elif action == "update_graph":
            attrs = params.get("attributes", {})
            for k, v in attrs.items():
                if v is None or v == "":
                    # Can't easily delete graph attrs in pydot, set to empty if needed
                    pass
                else:
                    graph.set(k, str(v))
        else:
            return {"success": False, "error": f"Unknown action: {action}"}

    except Exception as e:
        return {"success": False, "error": f"Error modifying graph: {str(e)}"}

    # Compile modified graph to string
    try:
        new_dot = graph.to_string()
    except Exception as e:
        return {"success": False, "error": f"Failed to serialize graph to DOT: {str(e)}"}

    # Render updated graph
    res = render_graphviz(new_dot, req.engine)
    if not res["success"]:
        # If it fails to render, return the DOT code anyway with the error
        return {"success": False, "error": res["error"], "dot_code": new_dot}
        
    return {
        "success": True,
        "dot_code": new_dot,
        "svg": res["svg"],
        "graph_data": res["graph_data"]
    }

# Mount static files directory
static_dir = os.path.join(os.path.dirname(__file__), "static")
os.makedirs(static_dir, exist_ok=True)
app.mount("/static", StaticFiles(directory=static_dir), name="static")

@app.get("/")
def read_root():
    index_path = os.path.join(static_dir, "index.html")
    if os.path.exists(index_path):
        return FileResponse(index_path)
    return {"message": "Dynamic Graphviz SVG Studio. Please create static/index.html"}

if __name__ == "__main__":
    import uvicorn
    # Use port 60002 as allocated in the implementation plan
    uvicorn.run("main:app", host="0.0.0.0", port=60002, reload=True)
