import sys
import os
from fastapi.testclient import TestClient

# Ensure the parent directory is in the path so we can import main
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from main import app

client = TestClient(app)

def test_read_root():
    response = client.get("/")
    assert response.status_code == 200

def test_api_render_valid():
    payload = {
        "dot_code": "digraph G { A -> B; }",
        "engine": "dot"
    }
    response = client.post("/api/render", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert "<svg" in data["svg"]
    assert data["graph_data"] is not None
    assert "objects" in data["graph_data"]
    # Check that nodes A and B exist in layout data
    nodes = [obj["name"] for obj in data["graph_data"]["objects"]]
    assert "A" in nodes
    assert "B" in nodes

def test_api_render_invalid():
    payload = {
        "dot_code": "digraph G { A -> B",  # Missing closing brace
        "engine": "dot"
    }
    response = client.post("/api/render", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is False
    assert "error" in data

def test_api_update_add_node():
    payload = {
        "dot_code": "digraph G { A -> B; }",
        "action": "add_node",
        "engine": "dot",
        "params": {
            "node_id": "C",
            "attributes": {
                "label": "Node C",
                "shape": "box",
                "fillcolor": "#ff0000"
            }
        }
    }
    response = client.post("/api/update", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert "C" in data["dot_code"]
    assert 'label="Node C"' in data["dot_code"]
    assert 'shape=box' in data["dot_code"]
    assert 'fillcolor="#ff0000"' in data["dot_code"]

def test_api_update_add_edge():
    payload = {
        "dot_code": "digraph G { A; B; }",
        "action": "add_edge",
        "engine": "dot",
        "params": {
            "source": "A",
            "target": "B",
            "attributes": {
                "label": "connects",
                "color": "#00ff00"
            }
        }
    }
    response = client.post("/api/update", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert "A -> B" in data["dot_code"]
    assert 'label=connects' in data["dot_code"]

def test_api_update_delete_node():
    # G has nodes A, B explicitly and edge A -> B. Deleting B should remove node B and edge A -> B, while A persists.
    payload = {
        "dot_code": "digraph G { A; B; A -> B; }",
        "action": "delete_node",
        "engine": "dot",
        "params": {
            "node_id": "B"
        }
    }
    response = client.post("/api/update", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    # The DOT code should contain A but not B or A -> B
    assert "A" in data["dot_code"]
    assert "B" not in data["dot_code"]


def test_api_update_update_node():
    payload = {
        "dot_code": "digraph G { A [label=\"Old Label\"]; }",
        "action": "update_node",
        "engine": "dot",
        "params": {
            "node_id": "A",
            "attributes": {
                "label": "New Label",
                "shape": "circle"
            }
        }
    }
    response = client.post("/api/update", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert 'label="New Label"' in data["dot_code"]
    assert 'shape=circle' in data["dot_code"]

def test_api_update_update_edge():
    payload = {
        "dot_code": "digraph G { A -> B [label=\"Old\"]; }",
        "action": "update_edge",
        "engine": "dot",
        "params": {
            "source": "A",
            "target": "B",
            "attributes": {
                "label": "New",
                "style": "dashed"
            }
        }
    }
    response = client.post("/api/update", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert 'label=New' in data["dot_code"]
    assert 'style=dashed' in data["dot_code"]
