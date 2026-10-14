# AutoMarket - Diagramas de Arquitectura (US-14)

Diagramas oficiales requeridos para la entrega final del taller de Ciberseguridad (US-14 y US-16).

---

## 1. Diagrama A: Estructura de Componentes y Paquetes (Go)

Arquitectura por capas, bajo acoplamiento y flujo de dependencias del backend:

```mermaid
flowchart TD
    Client(["Cliente HTTP"]) --> Gin["Gin Engine (github.com/gin-gonic/gin)"]

    subgraph Pipeline["Pipeline de Seguridad AAA (internal/auth)"]
        Audit["AuditLogMiddleware<br/><i>(Accounting)</i>"]
        Auth["AuthMiddleware<br/><i>(Authentication: Bearer)</i>"]
        RBAC["RequireRole<br/><i>(Authorization: RBAC)</i>"]
        Audit --> Auth --> RBAC
    end

    subgraph Handlers["internal/handlers"]
        H["Handlers HTTP<br/>• AuthHandler (Register, Login)<br/>• VehicleHandler (Create, MarkAsSold)<br/>• AdminHandler (Moderación, DeleteUser)<br/>• CatalogHandler (Catálogo público)"]
    end

    subgraph Stores["internal/store + auth (sync.RWMutex)"]
        direction LR
        US["UserStore"]
        VS["VehicleStore"]
        TS["TokenStore (TTL 24h)"]
    end

    subgraph Domain["internal/domain"]
        D["Entidades puras: User & VehiclePublication"]
    end

    Log[("audit.log<br/>(JSONL)")]

    Gin --> Audit
    RBAC --> H
    H --> Stores
    Stores --> Domain
    Audit -.->|Escritura concurrente| Log
    Auth -.->|Validación de sesión| TS
    H -.->|Hash SHA-256 + Salt| Crypto["crypto.go (stdlib)"]
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
    S-->>H: OK
    H-->>M: HTTP 201 Created (JSON)
    M-->>A: Retorno tras c.Next()
    A->>L: Escribe JSONL (latencia, user_id, role, status) [sync.Mutex]
    A-->>G: HTTP 201 Created
    G-->>V: HTTP 201 Created
```
