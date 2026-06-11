from fastapi import FastAPI, HTTPException, Depends, File, UploadFile, Form
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from sqlalchemy import create_engine, Column, Integer, String, DateTime, Text
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker, Session
from datetime import datetime
from typing import List, Optional
import os
import uuid
import shutil

# Database Setup
DATABASE_URL = "sqlite:///./wamr_logs.db"
engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()
 
# Database Model
class WasmLog(Base):
    __tablename__ = "wasm_logs"
    id = Column(Integer, primary_key=True, index=True)
    function_name = Column(String)
    input_data = Column(Text)
    output_data = Column(Text)
    file_path = Column(String, nullable=True)
    timestamp = Column(DateTime, default=datetime.utcnow)

Base.metadata.create_all(bind=engine)

# Pydantic Models
class LogCreate(BaseModel):
    function_name: str
    input_data: str
    output_data: str

class LogResponse(BaseModel):
    id: int
    function_name: str
    input_data: str
    output_data: str
    file_path: Optional[str]
    timestamp: datetime

    class Config:
        from_attributes = True

app = FastAPI(title="WAMR Output Logger")

# Ensure storage directory exists
UPLOAD_DIR = "downloads"
os.makedirs(UPLOAD_DIR, exist_ok=True)

# Serve static files
app.mount("/downloads", StaticFiles(directory=UPLOAD_DIR), name="downloads")

# Dependency
def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

@app.post("/log", response_model=LogResponse)
async def create_log(
    function_name: str = Form(...),
    input_data: str = Form(...),
    output_data: str = Form(...),
    file: UploadFile = File(None),
    db: Session = Depends(get_db)
):
    saved_file_path = None
    if file:
        file_extension = os.path.splitext(file.filename)[1]
        unique_filename = f"{uuid.uuid4()}{file_extension}"
        saved_file_path = os.path.join(UPLOAD_DIR, unique_filename)
        with open(saved_file_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)
        saved_file_path = f"/downloads/{unique_filename}"

    db_log = WasmLog(
        function_name=function_name,
        input_data=input_data,
        output_data=output_data,
        file_path=saved_file_path
    )
    db.add(db_log)
    db.commit()
    db.refresh(db_log)
    return db_log

@app.get("/logs", response_model=List[LogResponse])
def get_logs(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    logs = db.query(WasmLog).offset(skip).limit(limit).all()
    return logs

@app.get("/", response_class=HTMLResponse)
def read_root(db: Session = Depends(get_db)):
    logs = db.query(WasmLog).order_by(WasmLog.timestamp.desc()).all()
    
    log_rows = ""
    for log in logs:
        file_link = f'<a href="{log.file_path}" class="btn-download" target="_blank">📄 Download</a>' if log.file_path else '<span class="no-file">No File</span>'
        log_rows += f"""
        <tr>
            <td><span class="badge">#{log.id}</span></td>
            <td><strong>{log.function_name}</strong></td>
            <td><span class="time">{log.timestamp.strftime('%H:%M:%S')}<br><small>{log.timestamp.strftime('%Y-%m-%d')}</small></span></td>
            <td class="data-cell"><div class="data-scroll">{log.input_data}</div></td>
            <td class="data-cell"><div class="data-scroll">{log.output_data}</div></td>
            <td>{file_link}</td>
        </tr>
        """

    html_content = f"""
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>WAMR | Execution Dashboard</title>
        <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono&display=swap" rel="stylesheet">
        <style>
            :root {{
                --primary: #6366f1;
                --primary-dark: #4f46e5;
                --bg: #f8fafc;
                --card-bg: #ffffff;
                --text-main: #1e293b;
                --text-muted: #64748b;
                --border: #e2e8f0;
            }}
            * {{ box-sizing: border-box; }}
            body {{ 
                font-family: 'Inter', sans-serif; 
                margin: 0; 
                background: var(--bg); 
                color: var(--text-main);
                line-height: 1.5;
            }}
            .navbar {{
                background: white;
                border-bottom: 1px solid var(--border);
                padding: 1rem 2rem;
                display: flex;
                justify-content: space-between;
                align-items: center;
                position: sticky;
                top: 0;
                z-index: 100;
            }}
            .logo {{
                font-size: 1.25rem;
                font-weight: 700;
                color: var(--primary);
                display: flex;
                align-items: center;
                gap: 0.5rem;
            }}
            .container {{
                max-width: 1200px;
                margin: 2rem auto;
                padding: 0 1rem;
            }}
            .card {{
                background: var(--card-bg);
                border-radius: 12px;
                border: 1px solid var(--border);
                box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
                overflow: hidden;
            }}
            table {{
                width: 100%;
                border-collapse: collapse;
                text-align: left;
            }}
            th {{
                background: #f1f5f9;
                padding: 1rem;
                font-weight: 600;
                font-size: 0.875rem;
                color: var(--text-muted);
                text-transform: uppercase;
                letter-spacing: 0.05em;
            }}
            td {{
                padding: 1rem;
                border-bottom: 1px solid var(--border);
                vertical-align: middle;
            }}
            .badge {{
                background: #f1f5f9;
                padding: 0.25rem 0.5rem;
                border-radius: 6px;
                font-family: 'JetBrains Mono', monospace;
                font-size: 0.875rem;
                color: var(--text-muted);
            }}
            .time {{ font-size: 0.875rem; color: var(--text-muted); }}
            .data-cell {{ max-width: 300px; }}
            .data-scroll {{
                max-height: 60px;
                overflow-y: auto;
                font-family: 'JetBrains Mono', monospace;
                font-size: 0.75rem;
                background: #f8fafc;
                padding: 0.5rem;
                border-radius: 4px;
                white-space: pre-wrap;
                word-break: break-all;
            }}
            .btn-download {{
                display: inline-flex;
                align-items: center;
                padding: 0.5rem 0.75rem;
                background: var(--primary);
                color: white;
                text-decoration: none;
                border-radius: 6px;
                font-size: 0.875rem;
                font-weight: 500;
                transition: background 0.2s;
            }}
            .btn-download:hover {{ background: var(--primary-dark); }}
            .no-file {{ color: var(--text-muted); font-size: 0.875rem; font-style: italic; }}
            .stats {{
                display: grid;
                grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                gap: 1rem;
                margin-bottom: 2rem;
            }}
            .stat-card {{
                background: white;
                padding: 1.5rem;
                border-radius: 12px;
                border: 1px solid var(--border);
            }}
            .stat-value {{ font-size: 1.5rem; font-weight: 700; color: var(--primary); }}
            .stat-label {{ font-size: 0.875rem; color: var(--text-muted); }}
        </style>
    </head>
    <body>
        <nav class="navbar">
            <div class="logo">⚡ WAMR Explorer</div>
            <div class="status" style="font-size: 0.875rem; color: #10b981;">● Server Online</div>
        </nav>
        <div class="container">
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-value">{len(logs)}</div>
                    <div class="stat-label">Total Executions</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{len([l for l in logs if l.file_path])}</div>
                    <div class="stat-label">Files Generated</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{datetime.now().strftime('%H:%M')}</div>
                    <div class="stat-label">Last Updated</div>
                </div>
            </div>
            <div class="card">
                <table>
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Function</th>
                            <th>Time</th>
                            <th>Input Data</th>
                            <th>Output/Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {log_rows}
                    </tbody>
                </table>
            </div>
        </div>
    </body>
    </html>
    """
    return html_content

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
