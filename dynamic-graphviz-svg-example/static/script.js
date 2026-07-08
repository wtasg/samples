// Graphviz Flow - Interactive Client Script

document.addEventListener('DOMContentLoaded', () => {
    // --- Application State ---
    let state = {
        currentDot: '',
        layoutEngine: 'neato',
        selectedElement: null, // { type: 'node'|'edge', id: string, source?: string, target?: string }
        zoom: 1.0,
        pan: { x: 0, y: 0 },
        graphData: null,
        
        // Panning canvas state
        isPanning: false,
        panStart: { x: 0, y: 0 },
        
        // Drawing edge state
        isDrawingEdge: false,
        edgeStartNode: null,
        tempLine: null,
        
        // Dragging node state
        isDraggingNode: false,
        draggedNodeId: null,
        draggedNodeOriginalPos: null,
        dragMouseStart: { x: 0, y: 0 }
    };

    // --- DOM Elements ---
    const dotTextarea = document.getElementById('dot-textarea');
    const layoutEngineSelect = document.getElementById('layout-engine');
    const btnRender = document.getElementById('btn-render');
    const btnCopyDot = document.getElementById('btn-copy-dot');
    const btnDownloadSvg = document.getElementById('btn-download-svg');
    const btnClear = document.getElementById('btn-clear');
    const syncBadge = document.getElementById('sync-badge');
    const errorConsole = document.getElementById('error-console');
    const errorText = document.getElementById('error-text');
    const svgContainer = document.getElementById('svg-container');
    const canvasViewport = document.getElementById('canvas-viewport');
    
    // Zoom Buttons
    const btnZoomIn = document.getElementById('btn-zoom-in');
    const btnZoomOut = document.getElementById('btn-zoom-out');
    const btnZoomReset = document.getElementById('btn-zoom-reset');
    const btnZoomFit = document.getElementById('btn-zoom-fit');

    // Inspector forms
    const inspectorEmpty = document.getElementById('inspector-empty');
    const selectedTypeBadge = document.getElementById('selected-type-badge');
    const nodeForm = document.getElementById('inspector-node-form');
    const edgeForm = document.getElementById('inspector-edge-form');

    // Node Form inputs
    const inputNodeId = document.getElementById('node-id-display');
    const inputNodeLabel = document.getElementById('node-label');
    const inputNodeShape = document.getElementById('node-shape');
    const inputNodeFillcolor = document.getElementById('node-fillcolor');
    const inputNodeFillcolorText = document.getElementById('node-fillcolor-text');
    const inputNodeColor = document.getElementById('node-color');
    const inputNodeFontcolor = document.getElementById('node-fontcolor');
    const btnUpdateNode = document.getElementById('btn-update-node');
    const btnDeleteNode = document.getElementById('btn-delete-node');

    // Edge Form inputs
    const spanEdgeSource = document.getElementById('edge-source-display');
    const spanEdgeTarget = document.getElementById('edge-target-display');
    const inputEdgeLabel = document.getElementById('edge-label');
    const inputEdgeStyle = document.getElementById('edge-style');
    const inputEdgeColor = document.getElementById('edge-color');
    const btnUpdateEdge = document.getElementById('btn-update-edge');
    const btnDeleteEdge = document.getElementById('btn-delete-edge');

    // Toast
    const toast = document.getElementById('toast');

    // Tab buttons & panes
    const tabButtons = document.querySelectorAll('.tab-btn');
    const tabPanes = document.querySelectorAll('.tab-pane');

    // --- Debounced Auto Render ---
    let renderTimeout = null;
    dotTextarea.addEventListener('input', () => {
        setSyncStatus('working');
        clearTimeout(renderTimeout);
        renderTimeout = setTimeout(() => {
            triggerRender();
        }, 800);
    });

    // Keyboard shortcut (Ctrl+Enter to compile)
    window.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            e.preventDefault();
            triggerRender();
        }
    });

    // --- Tab Switching ---
    tabButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            tabButtons.forEach(b => b.classList.remove('active'));
            tabPanes.forEach(p => p.classList.remove('active'));
            
            btn.classList.add('active');
            const tabId = btn.getAttribute('data-tab');
            document.getElementById(tabId).classList.add('active');
        });
    });

    // --- Control Listeners ---
    layoutEngineSelect.addEventListener('change', (e) => {
        state.layoutEngine = e.target.value;
        triggerRender();
    });

    btnRender.addEventListener('click', () => {
        triggerRender();
    });

    btnCopyDot.addEventListener('click', () => {
        navigator.clipboard.writeText(dotTextarea.value).then(() => {
            showToast('DOT code copied to clipboard!');
        });
    });

    btnDownloadSvg.addEventListener('click', () => {
        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;
        
        // Clone and strip temporary interaction markers
        const clone = svgEl.cloneNode(true);
        clone.querySelectorAll('.connector-handle').forEach(h => h.remove());
        clone.querySelectorAll('.temp-connector-line').forEach(l => l.remove());
        
        const svgString = new XMLSerializer().serializeToString(clone);
        const blob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = `diagram_${state.layoutEngine}.svg`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        showToast('SVG exported successfully!');
    });

    btnClear.addEventListener('click', () => {
        if (confirm('Are you sure you want to clear the canvas? This resets the graph.')) {
            const defaultDot = `digraph G {
    node [style="filled", fillcolor="#2a2f3a", color="#4e5767", fontcolor="#ffffff", shape="ellipse"];
    edge [color="#636e72", fontcolor="#ffffff"];
}`;
            dotTextarea.value = defaultDot;
            deselectAll();
            triggerRender();
        }
    });

    // --- Helper Functions ---
    function setSyncStatus(status) {
        if (status === 'synced') {
            syncBadge.textContent = 'Synced';
            syncBadge.className = 'sync-badge synced';
        } else if (status === 'working') {
            syncBadge.textContent = 'Typing...';
            syncBadge.className = 'sync-badge working';
        } else if (status === 'rendering') {
            syncBadge.textContent = 'Rendering...';
            syncBadge.className = 'sync-badge working';
        }
    }

    function showToast(message) {
        toast.textContent = message;
        toast.classList.remove('hidden');
        setTimeout(() => {
            toast.classList.add('hidden');
        }, 2000);
    }

    function showError(errMessage) {
        errorConsole.classList.remove('hidden');
        errorText.textContent = errMessage;
    }

    function clearError() {
        errorConsole.classList.add('hidden');
        errorText.textContent = '';
    }

    // Convert Screen Coordinate (clientX, clientY) to SVG coordinate space
    function screenToSVG(svgEl, clientX, clientY) {
        const pt = svgEl.createSVGPoint();
        pt.x = clientX;
        pt.y = clientY;
        
        const graphGroup = svgEl.querySelector('#graph0') || svgEl;
        const ctm = graphGroup.getScreenCTM();
        if (!ctm) return { x: clientX, y: clientY };
        
        const transformed = pt.matrixTransform(ctm.inverse());
        return { x: transformed.x, y: transformed.y };
    }

    // Get the center coordinate of an SVG node group
    function getNodeCenter(nodeEl) {
        const ellipse = nodeEl.querySelector('ellipse');
        if (ellipse) {
            return { x: parseFloat(ellipse.getAttribute('cx')), y: parseFloat(ellipse.getAttribute('cy')) };
        }
        const circle = nodeEl.querySelector('circle');
        if (circle) {
            return { x: parseFloat(circle.getAttribute('cx')), y: parseFloat(circle.getAttribute('cy')) };
        }
        const rect = nodeEl.querySelector('rect');
        if (rect) {
            return {
                x: parseFloat(rect.getAttribute('x')) + parseFloat(rect.getAttribute('width')) / 2,
                y: parseFloat(rect.getAttribute('y')) + parseFloat(rect.getAttribute('height')) / 2
            };
        }
        const polygon = nodeEl.querySelector('polygon');
        if (polygon) {
            const pointsStr = polygon.getAttribute('points');
            if (pointsStr) {
                const points = pointsStr.trim().split(/\s+/).map(p => {
                    const [x, y] = p.split(',').map(parseFloat);
                    return { x, y };
                });
                let sx = 0, sy = 0;
                points.forEach(p => { sx += p.x; sy += p.y; });
                return { x: sx / points.length, y: sy / points.length };
            }
        }
        
        // Fallback using bounding box
        try {
            const bbox = nodeEl.getBBox();
            return { x: bbox.x + bbox.width / 2, y: bbox.y + bbox.height / 2 };
        } catch(e) {
            return { x: 0, y: 0 };
        }
    }

    // Parse clean name from SVG element title
    function getCleanName(titleText) {
        return titleText.trim().replace(/^"|"$/g, '');
    }

    // Parse edge source & target from edge title
    function parseEdgeTitle(titleText) {
        let source = '';
        let target = '';
        if (titleText.includes('->')) {
            const parts = titleText.split('->');
            source = getCleanName(parts[0]);
            target = getCleanName(parts[1]);
        } else if (titleText.includes('--')) {
            const parts = titleText.split('--');
            source = getCleanName(parts[0]);
            target = getCleanName(parts[1]);
        }
        return { source, target };
    }

    // Find node details in Graphviz JSON representation
    function findNodeInJson(nodeId) {
        if (!state.graphData || !state.graphData.objects) return null;
        return state.graphData.objects.find(obj => getCleanName(obj.name) === nodeId);
    }

    // Find edge details in Graphviz JSON representation
    function findEdgeInJson(sourceId, targetId) {
        if (!state.graphData || !state.graphData.edges || !state.graphData.objects) return null;
        
        // Find numeric internal gv IDs
        const srcObj = state.graphData.objects.find(o => getCleanName(o.name) === sourceId);
        const dstObj = state.graphData.objects.find(o => getCleanName(o.name) === targetId);
        
        if (!srcObj || !dstObj) return null;
        
        return state.graphData.edges.find(e => 
            (e.tail === srcObj._gvid && e.head === dstObj._gvid)
        );
    }

    // --- API Interactions ---
    function triggerRender() {
        setSyncStatus('rendering');
        const dotCode = dotTextarea.value;
        state.currentDot = dotCode;

        fetch('/api/render', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ dot_code: dotCode, engine: state.layoutEngine })
        })
        .then(res => res.json())
        .then(data => {
            if (data.success) {
                clearError();
                state.graphData = data.graph_data;
                renderSVG(data.svg);
                setSyncStatus('synced');
            } else {
                showError(data.error);
                setSyncStatus('synced');
            }
        })
        .catch(err => {
            showError('Server connection lost: ' + err.message);
            setSyncStatus('synced');
        });
    }

    function executeUpdate(action, params) {
        setSyncStatus('rendering');
        
        fetch('/api/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                dot_code: dotTextarea.value,
                action: action,
                engine: state.layoutEngine,
                params: params
            })
        })
        .then(res => res.json())
        .then(data => {
            if (data.success) {
                clearError();
                dotTextarea.value = data.dot_code;
                state.currentDot = data.dot_code;
                state.graphData = data.graph_data;
                renderSVG(data.svg);
                
                // Keep properties inspector updated or clear it
                if (state.selectedElement) {
                    if (action === 'delete_node' && params.node_id === state.selectedElement.id) {
                        deselectAll();
                    } else if (action === 'delete_edge' && 
                               params.source === state.selectedElement.source && 
                               params.target === state.selectedElement.target) {
                        deselectAll();
                    } else {
                        // Refresh selections
                        reselectElement();
                    }
                }
                setSyncStatus('synced');
            } else {
                showError(data.error);
                if (data.dot_code) {
                    dotTextarea.value = data.dot_code;
                }
                setSyncStatus('synced');
            }
        })
        .catch(err => {
            showError('Server connection lost: ' + err.message);
            setSyncStatus('synced');
        });
    }

    // --- Render and Add Interaction to SVG ---
    function renderSVG(svgString) {
        svgContainer.innerHTML = svgString;
        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;

        // Apply global styles to fit container
        svgEl.style.width = '100%';
        svgEl.style.height = '100%';
        
        // Force overflow visible for overlays/handles
        svgEl.style.overflow = 'visible';

        // Apply our current zoom/pan state to the SVG's root graph group
        const graphGroup = svgEl.querySelector('#graph0');
        if (graphGroup) {
            applyZoomPan(svgEl, graphGroup);
        }

        // Attach interaction to node elements
        const nodes = svgEl.querySelectorAll('.node');
        nodes.forEach(node => {
            const title = node.querySelector('title');
            if (!title) return;
            const nodeId = getCleanName(title.textContent);
            node.setAttribute('data-id', nodeId);

            // Click: select element
            node.addEventListener('click', (e) => {
                e.stopPropagation();
                selectNode(nodeId, node);
            });

            // Hover: Show connector handles & interaction overlays
            node.addEventListener('mouseenter', () => {
                showConnectorHandle(svgEl, node, nodeId);
            });

            node.addEventListener('mouseleave', (e) => {
                // If moving towards the handle, keep it, otherwise clean up
                const toEl = e.relatedTarget;
                if (toEl && toEl.classList.contains('connector-handle')) return;
                removeConnectorHandle(node);
            });

            // Drag and Drop (Node Relocation in neato)
            node.addEventListener('mousedown', (e) => {
                if (e.target.classList.contains('connector-handle')) return; // Handle connection trigger separately
                if (state.layoutEngine !== 'neato' && state.layoutEngine !== 'fdp') {
                    // Quick guide for user
                    return;
                }
                
                e.stopPropagation();
                e.preventDefault();
                
                const nodeCenter = getNodeCenter(node);
                
                state.isDraggingNode = true;
                state.draggedNodeId = nodeId;
                state.dragMouseStart = screenToSVG(svgEl, e.clientX, e.clientY);
                
                // Store original JSON position (which is in points: x,y)
                const jsonNode = findNodeInJson(nodeId);
                if (jsonNode && jsonNode.pos) {
                    const [x, y] = jsonNode.pos.split(',').map(parseFloat);
                    state.draggedNodeOriginalPos = { x, y };
                } else {
                    // Fallback to center computed from SVG geometry
                    state.draggedNodeOriginalPos = { x: nodeCenter.x, y: -nodeCenter.y };
                }

                node.classList.add('dragging');
            });
        });

        // Attach interaction to edge elements
        const edges = svgEl.querySelectorAll('.edge');
        edges.forEach(edge => {
            const title = edge.querySelector('title');
            if (!title) return;
            const { source, target } = parseEdgeTitle(title.textContent);
            edge.setAttribute('data-source', source);
            edge.setAttribute('data-target', target);

            edge.addEventListener('click', (e) => {
                e.stopPropagation();
                selectEdge(source, target, edge);
            });
        });

        // Click outside on canvas background: deselect
        svgEl.addEventListener('click', (e) => {
            if (e.target === svgEl || e.target.id === 'graph0' || e.target.tagName === 'polygon') {
                deselectAll();
            }
        });

        // Re-apply visual selection states if they still exist
        reselectElement();
    }

    // --- Zoom and Pan System ---
    function applyZoomPan(svgEl, graphGroup) {
        // Read original transform from Graphviz if it exists
        // (usually like: scale(1 1) rotate(0) translate(4 112))
        const origTransform = graphGroup.getAttribute('data-orig-transform') || graphGroup.getAttribute('transform') || '';
        if (!graphGroup.getAttribute('data-orig-transform')) {
            graphGroup.setAttribute('data-orig-transform', origTransform);
        }

        // Apply our interactive pan & zoom on top of the original layout matrix/translation
        svgEl.style.transform = `translate(${state.pan.x}px, ${state.pan.y}px) scale(${state.zoom})`;
        svgEl.style.transformOrigin = 'center center';
    }

    canvasViewport.addEventListener('mousedown', (e) => {
        // Only initiate panning if clicking background
        if (e.target === canvasViewport || e.target.id === 'svg-container' || e.target.tagName === 'svg' || e.target.id === 'graph0' || e.target.tagName === 'polygon') {
            state.isPanning = true;
            state.panStart = { x: e.clientX - state.pan.x, y: e.clientY - state.pan.y };
            canvasViewport.style.cursor = 'grabbing';
        }
    });

    window.addEventListener('mousemove', (e) => {
        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;

        // 1. Handle Canvas Panning
        if (state.isPanning) {
            state.pan.x = e.clientX - state.panStart.x;
            state.pan.y = e.clientY - state.panStart.y;
            const graphGroup = svgEl.querySelector('#graph0');
            if (graphGroup) applyZoomPan(svgEl, graphGroup);
        }

        // 2. Handle Node Reposition Dragging
        else if (state.isDraggingNode && state.draggedNodeId) {
            const svgPos = screenToSVG(svgEl, e.clientX, e.clientY);
            const dx = svgPos.x - state.dragMouseStart.x;
            const dy = svgPos.y - state.dragMouseStart.y;
            
            const nodeEl = svgEl.querySelector(`.node[data-id="${state.draggedNodeId}"]`);
            if (nodeEl) {
                // Instantly visual feedback by offset translation
                nodeEl.setAttribute('transform', `translate(${dx}, ${dy})`);
            }
        }

        // 3. Handle Edge Connection Line
        else if (state.isDrawingEdge && state.edgeStartNode) {
            const svgPos = screenToSVG(svgEl, e.clientX, e.clientY);
            if (state.tempLine) {
                state.tempLine.setAttribute('x2', svgPos.x);
                state.tempLine.setAttribute('y2', svgPos.y);
            }
        }
    });

    window.addEventListener('mouseup', (e) => {
        const svgEl = svgContainer.querySelector('svg');
        
        // 1. End Panning
        if (state.isPanning) {
            state.isPanning = false;
            canvasViewport.style.cursor = 'grab';
        }

        // 2. End Node Dragging (Update coordinates on server)
        else if (state.isDraggingNode && state.draggedNodeId && svgEl) {
            state.isDraggingNode = false;
            const nodeEl = svgEl.querySelector(`.node[data-id="${state.draggedNodeId}"]`);
            if (nodeEl) {
                nodeEl.classList.remove('dragging');
                nodeEl.removeAttribute('transform'); // Remove visual offset, server will re-render
            }
            
            const svgPos = screenToSVG(svgEl, e.clientX, e.clientY);
            const dx = svgPos.x - state.dragMouseStart.x;
            const dy = svgPos.y - state.dragMouseStart.y;

            if (Math.abs(dx) > 2 || Math.abs(dy) > 2) {
                // Compute new coordinates in Graphviz space (Y is inverted relative to SVG)
                const newX = Math.round(state.draggedNodeOriginalPos.x + dx);
                const newY = Math.round(state.draggedNodeOriginalPos.y - dy);
                
                executeUpdate('update_node', {
                    node_id: state.draggedNodeId,
                    attributes: {
                        pos: `${newX},${newY}!`
                    }
                });
            }
            state.draggedNodeId = null;
        }

        // 3. End Drawing Edge (Create connection)
        else if (state.isDrawingEdge && state.edgeStartNode && svgEl) {
            state.isDrawingEdge = false;
            
            // Clean up visual temp line
            if (state.tempLine) {
                state.tempLine.remove();
                state.tempLine = null;
            }

            // Find target node under pointer
            let targetNodeId = null;
            let el = document.elementFromPoint(e.clientX, e.clientY);
            
            // Traverse up to find a node element
            while (el && el !== document.body) {
                if (el.classList.contains('node')) {
                    targetNodeId = el.getAttribute('data-id');
                    break;
                }
                el = el.parentElement;
            }

            if (targetNodeId && targetNodeId !== state.edgeStartNode) {
                executeUpdate('add_edge', {
                    source: state.edgeStartNode,
                    target: targetNodeId,
                    attributes: {}
                });
            }
            state.edgeStartNode = null;
        }
    });

    // Zoom mouse wheel support
    canvasViewport.addEventListener('wheel', (e) => {
        e.preventDefault();
        const zoomIntensity = 0.08;
        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;

        if (e.deltaY < 0) {
            state.zoom = Math.min(state.zoom + zoomIntensity, 4.0);
        } else {
            state.zoom = Math.max(state.zoom - zoomIntensity, 0.2);
        }

        const graphGroup = svgEl.querySelector('#graph0');
        if (graphGroup) applyZoomPan(svgEl, graphGroup);
    });

    // Floating controls zoom logic
    btnZoomIn.addEventListener('click', () => {
        state.zoom = Math.min(state.zoom + 0.2, 4.0);
        const svgEl = svgContainer.querySelector('svg');
        if (svgEl) {
            const graphGroup = svgEl.querySelector('#graph0');
            if (graphGroup) applyZoomPan(svgEl, graphGroup);
        }
    });

    btnZoomOut.addEventListener('click', () => {
        state.zoom = Math.max(state.zoom - 0.2, 0.2);
        const svgEl = svgContainer.querySelector('svg');
        if (svgEl) {
            const graphGroup = svgEl.querySelector('#graph0');
            if (graphGroup) applyZoomPan(svgEl, graphGroup);
        }
    });

    btnZoomReset.addEventListener('click', () => {
        state.zoom = 1.0;
        state.pan = { x: 0, y: 0 };
        const svgEl = svgContainer.querySelector('svg');
        if (svgEl) {
            const graphGroup = svgEl.querySelector('#graph0');
            if (graphGroup) applyZoomPan(svgEl, graphGroup);
        }
    });

    btnZoomFit.addEventListener('click', () => {
        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;
        
        state.zoom = 0.95;
        state.pan = { x: 0, y: 0 };
        const graphGroup = svgEl.querySelector('#graph0');
        if (graphGroup) applyZoomPan(svgEl, graphGroup);
    });


    // --- Drag and Drop Connector Handle (Draw Edges) ---
    function showConnectorHandle(svgEl, nodeEl, nodeId) {
        // Check if handle already exists
        if (nodeEl.querySelector('.connector-handle')) return;

        const center = getNodeCenter(nodeEl);
        
        // Create SVG handle circle
        const handle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
        handle.setAttribute('class', 'connector-handle');
        handle.setAttribute('cx', center.x);
        handle.setAttribute('cy', center.y);
        handle.setAttribute('r', '5');
        handle.setAttribute('fill', '#3498db');
        handle.setAttribute('stroke', '#ffffff');
        handle.setAttribute('stroke-width', '1.5');
        
        // Listen to connection trigger
        handle.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            e.preventDefault();
            
            state.isDrawingEdge = true;
            state.edgeStartNode = nodeId;

            // Create temporary line for drawing
            const tempLine = document.createElementNS('http://www.w3.org/2000/svg', 'line');
            tempLine.setAttribute('class', 'temp-connector-line');
            tempLine.setAttribute('x1', center.x);
            tempLine.setAttribute('y1', center.y);
            tempLine.setAttribute('x2', center.x);
            tempLine.setAttribute('y2', center.y);
            
            const graphGroup = svgEl.querySelector('#graph0');
            if (graphGroup) {
                graphGroup.appendChild(tempLine);
                state.tempLine = tempLine;
            }
        });

        // Append handle to node group
        nodeEl.appendChild(handle);
    }

    function removeConnectorHandle(nodeEl) {
        const handle = nodeEl.querySelector('.connector-handle');
        if (handle && !state.isDrawingEdge) {
            handle.remove();
        }
    }


    // --- Drag and Drop Node Palette System ---
    const paletteItems = document.querySelectorAll('.palette-item');
    
    paletteItems.forEach(item => {
        item.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/plain', item.getAttribute('data-shape'));
            e.dataTransfer.setData('color', item.getAttribute('data-color'));
            e.dataTransfer.effectAllowed = 'move';
        });
    });

    canvasViewport.addEventListener('dragover', (e) => {
        e.preventDefault();
        canvasViewport.classList.add('drag-over');
    });

    canvasViewport.addEventListener('dragleave', () => {
        canvasViewport.classList.remove('drag-over');
    });

    canvasViewport.addEventListener('drop', (e) => {
        e.preventDefault();
        canvasViewport.classList.remove('drag-over');

        const shape = e.dataTransfer.getData('text/plain');
        const color = e.dataTransfer.getData('color') || '#3498db';
        if (!shape) return;

        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;

        // Get coordinates inside SVG space
        const svgPos = screenToSVG(svgEl, e.clientX, e.clientY);
        
        // Invert Y coordinate for Graphviz space
        const graphX = Math.round(svgPos.x);
        const graphY = Math.round(-svgPos.y);

        // Generate unique node ID
        let count = 1;
        let candidateId = `node${count}`;
        while (findNodeInJson(candidateId)) {
            count++;
            candidateId = `node${count}`;
        }

        // Trigger updates on backend
        executeUpdate('add_node', {
            node_id: candidateId,
            attributes: {
                label: `Node ${count}`,
                shape: shape,
                fillcolor: color,
                pos: `${graphX},${graphY}!` // Fixed coordinate marker
            }
        });
    });


    // --- Selection and Inspector Panel Updates ---
    function deselectAll() {
        state.selectedElement = null;
        
        const svgEl = svgContainer.querySelector('svg');
        if (svgEl) {
            svgEl.querySelectorAll('.node.selected').forEach(n => n.classList.remove('selected'));
            svgEl.querySelectorAll('.edge.selected').forEach(e => e.classList.remove('selected'));
        }

        selectedTypeBadge.textContent = 'None Selected';
        selectedTypeBadge.className = 'type-badge';
        nodeForm.classList.add('hidden');
        edgeForm.classList.add('hidden');
        inspectorEmpty.classList.remove('hidden');
    }

    function selectNode(nodeId, nodeEl) {
        deselectAll();
        state.selectedElement = { type: 'node', id: nodeId };
        nodeEl.classList.add('selected');

        selectedTypeBadge.textContent = 'Node';
        selectedTypeBadge.className = 'type-badge node-selected';
        inspectorEmpty.classList.add('hidden');
        edgeForm.classList.add('hidden');
        nodeForm.classList.remove('hidden');

        // Populate Form
        inputNodeId.value = nodeId;
        
        const jsonNode = findNodeInJson(nodeId);
        if (jsonNode) {
            inputNodeLabel.value = jsonNode.label || nodeId;
            inputNodeShape.value = jsonNode.shape || 'ellipse';
            
            // Read background color
            const fillcolor = jsonNode.fillcolor || '#2a2f3a';
            inputNodeFillcolor.value = fillcolor.startsWith('#') ? fillcolor : '#2a2f3a';
            inputNodeFillcolorText.value = fillcolor;
            
            const bordercolor = jsonNode.color || '#4e5767';
            inputNodeColor.value = bordercolor.startsWith('#') ? bordercolor : '#4e5767';
            
            const fontcolor = jsonNode.fontcolor || '#ffffff';
            inputNodeFontcolor.value = fontcolor.startsWith('#') ? fontcolor : '#ffffff';
        } else {
            inputNodeLabel.value = nodeId;
            inputNodeShape.value = 'ellipse';
            inputNodeFillcolor.value = '#2a2f3a';
            inputNodeFillcolorText.value = '#2a2f3a';
            inputNodeColor.value = '#4e5767';
            inputNodeFontcolor.value = '#ffffff';
        }
    }

    function selectEdge(sourceId, targetId, edgeEl) {
        deselectAll();
        state.selectedElement = { type: 'edge', id: `${sourceId}->${targetId}`, source: sourceId, target: targetId };
        edgeEl.classList.add('selected');

        selectedTypeBadge.textContent = 'Edge';
        selectedTypeBadge.className = 'type-badge edge-selected';
        inspectorEmpty.classList.add('hidden');
        nodeForm.classList.add('hidden');
        edgeForm.classList.remove('hidden');

        // Populate Form
        spanEdgeSource.textContent = sourceId;
        spanEdgeTarget.textContent = targetId;

        const jsonEdge = findEdgeInJson(sourceId, targetId);
        if (jsonEdge) {
            inputEdgeLabel.value = jsonEdge.label || '';
            inputEdgeStyle.value = jsonEdge.style || 'solid';
            const edgeColor = jsonEdge.color || '#636e72';
            inputEdgeColor.value = edgeColor.startsWith('#') ? edgeColor : '#636e72';
        } else {
            inputEdgeLabel.value = '';
            inputEdgeStyle.value = 'solid';
            inputEdgeColor.value = '#636e72';
        }
    }

    // Refresh visual highlights if active selection still exists in new SVG
    function reselectElement() {
        if (!state.selectedElement) return;

        const svgEl = svgContainer.querySelector('svg');
        if (!svgEl) return;

        if (state.selectedElement.type === 'node') {
            const nodeEl = svgEl.querySelector(`.node[data-id="${state.selectedElement.id}"]`);
            if (nodeEl) {
                nodeEl.classList.add('selected');
                selectNode(state.selectedElement.id, nodeEl);
            } else {
                deselectAll();
            }
        } else if (state.selectedElement.type === 'edge') {
            const { source, target } = state.selectedElement;
            const edgeEl = svgEl.querySelector(`.edge[data-source="${source}"][data-target="${target}"]`);
            if (edgeEl) {
                edgeEl.classList.add('selected');
                selectEdge(source, target, edgeEl);
            } else {
                deselectAll();
            }
        }
    }

    // Connect text input and picker color sync
    inputNodeFillcolor.addEventListener('input', (e) => {
        inputNodeFillcolorText.value = e.target.value;
    });
    inputNodeFillcolorText.addEventListener('input', (e) => {
        if (e.target.value.startsWith('#') && e.target.value.length === 7) {
            inputNodeFillcolor.value = e.target.value;
        }
    });

    // Inspector Action Listeners
    btnUpdateNode.addEventListener('click', () => {
        if (!state.selectedElement || state.selectedElement.type !== 'node') return;
        
        executeUpdate('update_node', {
            node_id: state.selectedElement.id,
            attributes: {
                label: inputNodeLabel.value,
                shape: inputNodeShape.value,
                fillcolor: inputNodeFillcolorText.value,
                color: inputNodeColor.value,
                fontcolor: inputNodeFontcolor.value
            }
        });
    });

    btnDeleteNode.addEventListener('click', () => {
        if (!state.selectedElement || state.selectedElement.type !== 'node') return;
        if (confirm(`Are you sure you want to delete node "${state.selectedElement.id}"? This also removes its connections.`)) {
            executeUpdate('delete_node', {
                node_id: state.selectedElement.id
            });
        }
    });

    btnUpdateEdge.addEventListener('click', () => {
        if (!state.selectedElement || state.selectedElement.type !== 'edge') return;
        
        executeUpdate('update_edge', {
            source: state.selectedElement.source,
            target: state.selectedElement.target,
            attributes: {
                label: inputEdgeLabel.value,
                style: inputEdgeStyle.value,
                color: inputEdgeColor.value
            }
        });
    });

    btnDeleteEdge.addEventListener('click', () => {
        if (!state.selectedElement || state.selectedElement.type !== 'edge') return;
        if (confirm(`Are you sure you want to delete the connection from "${state.selectedElement.source}" to "${state.selectedElement.target}"?`)) {
            executeUpdate('delete_edge', {
                source: state.selectedElement.source,
                target: state.selectedElement.target
            });
        }
    });

    // Keyboard 'Delete' support for active selection
    window.addEventListener('keydown', (e) => {
        if (e.key === 'Delete') {
            // Check if user is typing in a form input
            if (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA') return;

            if (state.selectedElement) {
                if (state.selectedElement.type === 'node') {
                    btnDeleteNode.click();
                } else if (state.selectedElement.type === 'edge') {
                    btnDeleteEdge.click();
                }
            }
        }
    });

    // --- Init App ---
    triggerRender();
});
