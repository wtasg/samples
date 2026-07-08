import subprocess
import time
import socket
import pytest

def is_port_open(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        return s.connect_ex(('127.0.0.1', port)) == 0

@pytest.fixture(scope="session", autouse=True)
def run_server():
    port = 60002
    server_url = f"http://127.0.0.1:{port}"
    
    # If server is already running, reuse it
    if is_port_open(port):
        yield server_url
        return

    # Start the FastAPI server in a subprocess
    process = subprocess.Popen(
        ["./venv/bin/python", "main.py"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    # Wait for the server to start accepting connections (up to 5 seconds)
    started = False
    for _ in range(50):
        if is_port_open(port):
            started = True
            break
        time.sleep(0.1)

    if not started:
        process.terminate()
        raise RuntimeError("Failed to start FastAPI server for testing")

    yield server_url

    # Clean up the server after testing
    process.terminate()
    try:
        process.wait(timeout=3)
    except subprocess.TimeoutExpired:
        process.kill()
