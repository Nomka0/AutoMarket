# AutoMarket - Diagramas de Arquitectura (US-14)

Diagramas oficiales requeridos para la entrega final del taller de Ciberseguridad (US-14 y US-16).

---

## 1. Diagrama A: Estructura de Componentes y Paquetes (Go)

Arquitectura por capas, bajo acoplamiento y flujo de dependencias del backend:

```mermaid
flowchart TD
    Client(["Cliente HTTP"]) --> Gin["Gin Engine (Única dependencia)"]

    subgraph Pipeline["Pipeline de Seguridad AAA (internal/auth)"]
        direction LR
        Audit["AuditLogMiddleware<br/><i>(Accounting)</i>"] --> Auth["AuthMiddleware<br/><i>(Authentication)</i>"] --> RBAC["RequireRole<br/><i>(Authorization)</i>"]
    end

    subgraph Core["Capa Aplicación y Persistencia (internal/)"]
        direction LR
        H["Handlers HTTP<br/>(Auth, Vehicle, Admin, Catalog)"]
        S["Stores (sync.RWMutex + JSON)<br/>UserStore | VehicleStore | TokenStore"]
        D["Domain Models<br/>User | VehiclePublication"]
        H --> S --> D
    end

    Gin --> Audit
    RBAC --> H
    Audit -.-> Log[("audit.log (JSONL)")]
    S -.-> Files[("Persistencia JSON (stdlib)<br/>data/users.json<br/>data/vehicles.json")]
    H -.-> Crypto["crypto.go (stdlib)"]
```

---

## 2. Diagrama B: Comportamiento (Secuencia AAA)

Traza de una petición protegida (`POST /api/seller/vehicles`) a través del marco de seguridad AAA:

```mermaid
sequenceDiagram
    autonumber
    actor V as Vendedor
    participant G as Gin Router
    participant A as AuditLog (Accounting)
    participant M as Auth & RBAC (AuthN/AuthZ)
    participant H as VehicleHandler
    participant S as VehicleStore (sync.RWMutex)
    participant L as audit.log

    V->>G: POST /api/seller/vehicles (Bearer token)
    G->>A: Inicia interceptor (start = time.Now)
    A->>M: c.Next()
    M->>M: Valida Bearer token en TokenStore (200 / 401)
    M->>M: Valida rol == 'vendedor' (200 / 403)
    M->>H: CreatePublication(c) con currentUser
    H->>S: Create(pub) con Lock() [status: PENDING]
    Note over S: save() vuelca a data/vehicles.json
    S-->>H: OK
    H-->>M: HTTP 201 Created (JSON)
    M-->>A: Retorno tras c.Next()
    A->>L: Escribe JSONL (latencia, user_id, role, status) [sync.Mutex]
    A-->>G: HTTP 201 Created
    G-->>V: HTTP 201 Created
```
