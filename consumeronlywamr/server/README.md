# WAMR FastAPI Logger

This is a small FastAPI server designed to log the inputs and outputs of WASM functions executed within the WAMR Sandbox app.

## Setup

1.  **Navigate to the server directory**:
    ```bash
    cd server
    ```

2.  **Create a virtual environment (optional but recommended)**:
    ```bash
    python -m venv venv
    source venv/bin/activate  # On macOS/Linux
    # venv\Scripts\activate  # On Windows
    ```

3.  **Install dependencies**:
    ```bash
    pip install -r requirements.txt
    ```

4.  **Run the server**:
    ```bash
    uvicorn main:app --host 0.0.0.0 --port 8000 --reload
    ```

## Endpoints

-   `POST /log`: Send a log entry.
    -   Body: `{"function_name": "...", "input_data": "...", "output_data": "..."}`
-   `GET /logs`: Retrieve all logged entries.
-   `GET /`: Root health check.

## Connecting from Mobile/Emulator

-   **Android Emulator**: Use `http://10.0.2.2:8000`.
-   **Physical Device**: Use your computer's local IP (e.g., `http://192.168.1.x:8000`). Make sure both devices are on the same network and firewall allows port 8000.
