# 📒 Enterprise Khata Book - Hierarchical Organization Ledger

A production-ready Enterprise Financial Khata Book built with **Go (1.22+)** and **SQLite** using **Domain-Driven Design (DDD)**. Designed for multi-department organizations to track financial transactions (Jama/Credits and Udhar/Debits) across a 5-level deep hierarchy tree (Department → Group → Subgroups) with real-time consolidated roll-up totals, cash flow analytics, and CSV streaming exports.

---

## 🏛️ Domain-Driven Design (DDD) Architecture

The codebase is organized into cleanly decoupled layers:

```
├── models/                     # Domain Entities, Enums, Money/Paise Helpers & Errors
│   ├── models.go               # Node, Entry, User, Department, RollUpTotals, Filters
│   ├── money.go                # Integer Paise <-> Float Rupees conversions & Indian formatting
│   └── errors.go               # Domain errors (ErrMaxDepthExceeded, ErrMoveExceedsMaxDepth, etc.)
│
├── repository/                 # Data Access, Persistence & Concurrency
│   ├── db.go                   # SQLite connection, WAL mode PRAGMAs, migrations & seed
│   ├── write_queue.go          # Go channel-based worker pool serializing all writes
│   ├── node_repo.go            # Materialized Path + Adjacency List tree queries & atomic move
│   ├── entry_repo.go           # SQL-level pagination, search & date range filters
│   ├── analytics_repo.go       # Path prefix rollups, monthly trends & macro overview
│   └── user_repo.go            # RBAC user & department scope queries
│
├── services/                   # Business Logic & Invariant Enforcement
│   ├── khata_service.go        # MAX_DEPTH = 5 validation, cycle prevention, RBAC checks
│   └── khata_service_test.go   # Unit test suite
│
├── handlers/                   # Standard Library net/http HTTP Handlers
│   ├── nodes.go                # Node CRUD & atomic tree moves
│   ├── entries.go              # Entry CRUD, search, date filtering & pagination
│   ├── analytics.go            # Consolidated roll-up, monthly trends, macro overview
│   ├── export.go               # Direct HTTP streaming CSV download
│   ├── middleware.go           # RBAC authentication & error handlers
│   ├── routes.go               # Route registration
│   └── handlers_test.go        # HTTP API integration tests
│
├── cmd/server/
│   └── main.go                 # Application entrypoint & dependency injection
│
└── web/                        # Modern Fintech Web UI
    ├── index.html              # Responsive single-page interface
    └── static/
        ├── css/style.css       # Clean fintech styling with Dark/Light themes
        └── js/app.js           # Client-side reactivity & SVG cash flow trends
```

---

## ⚡ Concurrency & SQLite WAL Mode

SQLite is an in-process database that supports multiple concurrent readers but only one concurrent writer. Under concurrent multi-user load, concurrent writes can trigger `database is locked` errors.

To guarantee zero database locking errors:
1. **SQLite WAL Mode (`PRAGMA journal_mode = WAL;`)**: Concurrent readers (`SELECT`) never block and run in parallel using SQLite's connection pool.
2. **Channel-Based Write Worker Pool (`repository/write_queue.go`)**:
   - All write operations (`INSERT`, `UPDATE`, `DELETE`) are packaged into `WriteTask` structs and dispatched via a buffered Go channel (`taskChan chan WriteTask`).
   - A single dedicated worker goroutine consumes tasks from the channel, opens an atomic transaction (`tx, err := wq.db.Begin()`), runs the write closure, commits, and returns errors over a completion channel (`ErrChan chan error`).
   - This ensures strictly serialized writes with zero contention and zero lock timeouts.

---

## 🌲 Hierarchical Data Model: Materialized Path + Adjacency List

The hierarchy tree combines **Materialized Paths** with an **Adjacency List**:
- `parent_id REFERENCES ledger_nodes(id) ON DELETE CASCADE`
- `path TEXT NOT NULL` (e.g. `/dept-fin/group-1/sub-2/`)
- `depth INTEGER NOT NULL CHECK(depth >= 1 AND depth <= 5)`

### Why Hybrid Path + Adjacency List?
1. **$O(1)$ Descendant Queries**: Any descendant of node $N$ satisfies `WHERE path LIKE N.Path || '%'`. This allows instant rollups across arbitrarily deep subtrees without recursive CTE overhead.
2. **$O(1)$ Ancestor Queries**: Upward breadcrumbs are fetched using `WHERE targetPath LIKE path || '%' ORDER BY depth ASC`.
3. **Atomic Subtree Relocation**: Moving node $A$ (depth $d_A$, old path $P_{old}$) to parent $B$ (depth $d_B$, path $P_{new}$) is executed in a single atomic SQL statement:
   ```sql
   UPDATE ledger_nodes
   SET path = ? || SUBSTR(path, LENGTH(?) + 1),
       depth = depth + ?,
       department_id = ?,
       updated_at = CURRENT_TIMESTAMP
   WHERE path LIKE ? || '%';
   ```
4. **Strict `MAX_DEPTH = 5`**:
   Before a move is executed, the service verifies that $(d_B + 1) + (\max(\text{depth in } A\text{'s subtree}) - d_A) \le 5$. If it would exceed 5, the move is rejected with `ErrMoveExceedsMaxDepth`.

---

## 💰 Integer Paise Financial Accuracy

To avoid IEEE 754 floating-point rounding errors (where $0.1 + 0.2 \ne 0.3$), financial amounts are stored as **integer Paise** (cents):
- `amount_paise INTEGER NOT NULL CHECK (amount_paise > 0)`
- 1 Indian Rupee (₹) = 100 Paise.
- Formatted in the UI using Indian numbering: e.g. `₹ 1,50,000.00` (Lakhs & Crores).

---

## 🛡️ Authentication & Role-Based Access Control (RBAC)

The system supports two enterprise roles:
1. **Global Admin (`Alice Sharma`)**: Full organization-wide access across all departments.
2. **Department Viewer (`Bob Verma`, `Charlie Patel`, `Diana Rao`)**: Scoped strictly to their assigned departments via `user_departments`.

Unauthorized access returns **HTTP 403 Forbidden**.

---

## 📡 Core API Surface

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/overview` | `GET` | Macro overview with org-wide totals and department rollups |
| `/api/departments` | `GET` | List departments visible to the user |
| `/api/departments/{deptID}/tree` | `GET` | Hierarchical tree with nested rollup metrics |
| `/api/nodes/{id}` | `GET` | Node details, breadcrumb path, and rollup |
| `/api/nodes` | `POST` | Create department (depth 1) or group/subgroup (depth 2..5) |
| `/api/nodes/{id}/rename` | `PUT` | Rename a node |
| `/api/nodes/{id}/move` | `POST` | Move a node and its subtree with depth limit validation |
| `/api/nodes/{id}` | `DELETE` | Delete a node and cascade-delete its subtree |
| `/api/nodes/{id}/entries` | `GET` | List entries with pagination (`page`, `limit`), date filter, search |
| `/api/nodes/{id}/entries` | `POST` | Record a new Khata entry (CREDIT/Jama or DEBIT/Udhar) |
| `/api/entries/{id}` | `DELETE` | Delete an entry |
| `/api/entries/bulk-delete` | `POST` | Bulk delete selected transactions |
| `/api/entries/bulk-move` | `POST` | Bulk move transactions to another node |
| `/api/nodes/{id}/rollup` | `GET` | Consolidated financial rollups using materialized path |
| `/api/nodes/{id}/trends` | `GET` | Monthly cash flow time-series (Jama vs Udhar) |
| `/api/nodes/{id}/export` | `GET` | Direct HTTP streaming `.csv` download |

---

## 🧪 Running Tests

The test suite validates depth limit enforcement, subtree move depth validation, cycle detection, roll-up mathematics, RBAC scoping, and HTTP endpoints:

```powershell
go test -v ./...
```

---

## 🚀 Running the Application

```powershell
go run ./cmd/server
```

Open your browser at:
👉 **`http://localhost:8080`**
