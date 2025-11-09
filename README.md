# QueueCTL — Flam Assignment

-> [Video](https://drive.google.com/file/d/1bdB35xgUtMrD1LjNe0cf1GDR3DtgtvBo/view?usp=sharing)
: https://drive.google.com/file/d/1bdB35xgUtMrD1LjNe0cf1GDR3DtgtvBo/view?usp=sharing 
---

## 1. Setup Instructions

### Prerequisites
- Go 1.20+
- SQLite (no installation required; DB auto-creates)

### Clone & Build

```bash
git clone https://github.com/navjo3/queuectl
cd queuectl
go build -o queuectl ./cmd/queuectl
```

### (Optional) Add to PATH

### Windows:

#### Adding queuectl to PATH (Windows Recommended Setup)
```powershell
mkdir C:\Users\%USERNAME%\bin

```
Move the binary into it:
```
move queuectl.exe C:\Users\%USERNAME%\bin\
```
Add the directory to PATH:
```
Open Start → search: Edit the system environment variables
Click Environment Variables
Under User Variables, select Path → Edit
Click New → add:
C:\Users\%USERNAME%\bin
```

Verify:
```
queuectl --help
```

#### ### Or Easy way
```powershell
move queuectl.exe C:\Windows\System32\

```

### Linux/macOS:
```bash
sudo mv queuectl /usr/local/bin/
```

Verify installation:

```bash
queuectl --help
```

---

## 2. Database Schema

### **Jobs Table**

| Column | Type | Description |
|-------|------|-------------|
| id | TEXT PRIMARY KEY | Unique job ID |
| command | TEXT | Shell command executed by the worker |
| state | ENUM | (`pending`, `processing`, `completed`, `dead`) |
| attempts | INTEGER | Number of execution attempts |
| max_retries | INTEGER | Retry limit before moving to DLQ |
| created_at | TEXT | Timestamp created |
| updated_at | TEXT | Last update timestamp |
| available_at | TEXT | When the job becomes eligible to run |

### **DLQ Table**

| Column | Type | Description |
|-------|------|-------------|
| id | TEXT PRIMARY KEY | Job identifier |
| command | TEXT | Original job command |
| attempts | INTEGER | Final attempts count |
| max_retries | INTEGER | Retry limit originally set |
| created_at | TEXT | Original creation timestamp |
| updated_at | TEXT | Last failure timestamp |
| failed_at | TEXT | Time job entered DLQ |

### **Config Table**

| Key | Description |
|-----|-------------|
| max_retries | Default retry limit |
| backoff_base | Exponential retry growth (e.g., 2 = 2^attempts) |
| backoff_cap_seconds | Maximum backoff delay in seconds |

---

## 3. Usage Examples

### Enqueue Jobs
```bash
queuectl enqueue '{"id":"job1","command":"echo Hello"}'
queuectl enqueue '{"id":"job2","command":"sleep 2"}'
```

### List Jobs
```bash
queuectl list
```

### Start Workers
```bash
queuectl worker start --count 2
```

### Stop Workers Gracefully
```bash
queuectl worker stop
```

### Queue Status
```bash
queuectl status
```

### Dead Letter Queue
```bash
queuectl dlq list
queuectl dlq retry <jobID>
```

### Change Configuration
```bash
queuectl config set max_retries 5
queuectl config set backoff_base 3
queuectl config set backoff_cap_seconds 90
```

### Reset Queue (Development Only) : To reset the created tables.
```bash
queuectl reset
```

---

## 4. Architecture Overview
![Architecture --Excalidraw](assets\architecture.png)
```

queuectl enqueue
        ↓
 +---------------+
 |   jobs table  |
 +-------+-------+
         |
         ↓ (workers select pending jobs)
 +--------------------------+
 |   Worker Engine Loop     |
 |--------------------------|
 | Fetch pending job        |
 | Mark as processing       |
 | Execute command          |
 | If success → completed   |
 | If fail → attempts++     |
 | Compute retry delay      |
 |   delay = min(base^attempts, cap) |
 | Reschedule (available_at) |
 +--------------------------+
         |
         ↓ (if attempts >= max_retries)
 +----------------+
 |   DLQ table    |
 +----------------+
```

---

## 5. Assumptions & Trade-offs

| Decision | Reason | Trade-off |
|---------|--------|----------|
| SQLite storage | Simple + durable + single binary deploy | Not designed for extreme concurrency throughput |
| Exponential backoff | Prevents retry storms + CPU thrashing | High attempts → long delays |
| DLQ instead of infinite retry | Prevents lockup under failure | Requires manual inspection & retry |
| Graceful shutdown | Safe job consistency | Slight delay when stopping workers |

---

## 6. Testing Instructions

### Run Automated Tests
```bash
go test ./... -v
```

Tests cover: Under internal/tests
- Enqueueing and job persistence
- Worker execution flow
- Retry + backoff timing
- DLQ migration logic
- DLQ retry recovery
- Config-driven behavior

### Manual Test Example
```bash
queuectl reset
queuectl enqueue '{"id":"demo","command":"echo hi"}'
queuectl worker start --count 1
queuectl worker stop
queuectl list
```

Expected Output:
```
demo | completed | attempts=0/3 | echo hi
```

---

## Navjyoth Pradeep
