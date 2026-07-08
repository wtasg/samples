import pytest
import re
from playwright.sync_api import Page, expect

def test_e2e_load_page_and_layout(page: Page, run_server):
    # Navigate to the application
    page.goto(run_server)
    
    # Verify page title and header
    expect(page).to_have_title("Graphviz Flow - Interactive SVG Diagram Studio")
    expect(page.locator("h1")).to_have_text("Graphviz Flow")
    
    # Verify the code editor contains initial DOT script
    textarea = page.locator("#dot-textarea")
    expect(textarea).to_be_visible()
    expect(textarea).to_have_value(re.compile(r"digraph G"))
    expect(textarea).to_have_value(re.compile(r"Start Node"))
    
    # Wait for the canvas to render the SVG
    svg_container = page.locator("#svg-container")
    expect(svg_container).to_be_visible()
    
    svg = svg_container.locator("svg")
    expect(svg).to_be_visible()
    
    # Verify initial nodes A, B, and C exist in SVG structure
    expect(svg.locator('.node[data-id="A"]')).to_be_visible()
    expect(svg.locator('.node[data-id="B"]')).to_be_visible()
    expect(svg.locator('.node[data-id="C"]')).to_be_visible()


def test_e2e_inspector_selection_and_edit(page: Page, run_server):
    page.goto(run_server)
    
    # Select Node A ("Start Node") in the SVG diagram
    node_a = page.locator('.node[data-id="A"]')
    node_a.click()
    
    # Verify properties panel is populated with Node info
    badge = page.locator("#selected-type-badge")
    expect(badge).to_have_text("Node")
    expect(badge).to_have_class("type-badge node-selected")
    
    expect(page.locator("#node-id-display")).to_have_value("A")
    
    node_label_input = page.locator("#node-label")
    expect(node_label_input).to_have_value("Start Node")
    
    # Edit label
    node_label_input.fill("Main Entrance")
    
    # Select Box shape
    page.select_option("#node-shape", "box")
    
    # Click Apply Changes
    page.click("#btn-update-node")
    
    # Verify editor window synced the updated code
    textarea = page.locator("#dot-textarea")
    expect(textarea).to_have_value(re.compile(r"Main Entrance"))
    expect(textarea).to_have_value(re.compile(r"shape=box"))
    
    # Verify live SVG renders the new label and shape
    expect(node_a.locator("text")).to_have_text("Main Entrance")
    expect(node_a.locator("polygon")).to_be_visible() # Graphviz box is rendered as polygon in SVG


def test_e2e_tab_switching_and_palette(page: Page, run_server):
    page.goto(run_server)
    
    # Verify initial tab is active
    expect(page.locator('#tab-editor')).to_be_visible()
    
    # Switch to Shape Palette tab
    page.click('.tab-btn:has-text("Shape Palette")')
    
    # Verify palette pane is displayed and editor hidden
    expect(page.locator('#tab-editor')).not_to_be_visible()
    expect(page.locator('#tab-palette')).to_be_visible()
    
    # Verify shape templates exist in palette
    expect(page.locator('.palette-item:has-text("Rectangle Node")')).to_be_visible()
    expect(page.locator('.palette-item:has-text("Ellipse Node")')).to_be_visible()
    expect(page.locator('.palette-item:has-text("Diamond Node")')).to_be_visible()
