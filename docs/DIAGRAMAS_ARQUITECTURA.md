# AutoMarket - Diagramas de Arquitectura (US-14)

Este documento contiene los dos diagramas oficiales requeridos para la entrega final del taller de Ciberseguridad (US-14 y US-16):
1. **Diagrama A: Diagrama de Estructura** (Componentes y paquetes de Go).
2. **Diagrama B: Diagrama de Comportamiento** (Secuencia del flujo de seguridad AAA).

---

## 1. Diagrama A: Diagrama de Estructura (Componentes y Paquetes Go)

Ilustra la arquitectura por capas, la separación de responsabilidades, la inyección de dependencias y el bajo acoplamiento entre los paquetes del proyecto.

```mermaid
flowchart TD
    subgraph CMD["cmd/server"]
        Main["main.go<br/>(Arranque, Inyección de Dependencias y Enrutamiento)"]
    end

    subgraph Framework["Dependencia Externa Única"]
        Gin["Gin Web Engine<br/><i>(github.com/gin-gonic/gin)</i>"]
    end

    subgraph Handlers["internal/handlers (Adaptadores HTTP)"]
        AuthH["AuthHandler<br/>• Register<br/>• Login"]
        VehH["VehicleHandler<br/>• CreatePublication<br/>• MarkAsSold"]
        AdminH["AdminHandler<br/>• GetPendingPublications<br/>• ApprovePublication<br/>• DeletePublication<br/>• DeleteUser"]
        CatH["CatalogHandler<br/>• GetCatalog<br/>• GetCatalogDetail"]
    end

    subgraph Auth["internal/auth (Seguridad y Marco AAA)"]
        Crypto["crypto.go<br/>• HashPassword / VerifyPassword<br/>• GenerateSalt / GenerateToken<br/><i>(crypto/sha256 + crypto/rand stdlib)</i>"]
        TokStore["TokenStore<br/>• Sesiones en memoria con TTL 24h<br/>• ValidateToken / DeleteByUserID"]
        AuditLogMid["AuditLogMiddleware<br/><i>(Accounting trazable e íntegro)</i>"]
        AuthMid["AuthMiddleware<br/><i>(Authentication Bearer Token)</i>"]
        RoleMid["RequireRole<br/><i>(Authorization RBAC)</i>"]
    end

    subgraph Storage["internal/store (Almacenamiento en Memoria Concurrente)"]
        UStore["UserStore<br/>• map[string]User<br/>• sync.RWMutex<br/>• SeedAdmin"]
        VStore["VehicleStore<br/>• map[string]VehiclePublication<br/>• sync.RWMutex"]
    end

    subgraph Domain["internal/domain (Entidades y Reglas del Dominio)"]
        UserDom["User<br/>(ID, Username, PasswordHash, Salt, Role, ContactInfo)"]
        VehDom["VehiclePublication<br/>(ID, SellerID, SellerContact, Title, Description, Price, Year, Status)"]
    end

    subgraph Filesystem["Persistencia en Disco"]
        AuditFile[("audit.log<br/>(Registros JSON Lines cronológicos)")]
    end

    %% Relaciones de inicialización
    Main --> Gin
    Main --> Auth
    Main --> Handlers
    Main --> Storage

    %% Pipeline de ejecución HTTP
    Gin --> AuditLogMid
    AuditLogMid --> AuthMid
    AuthMid --> RoleMid
    RoleMid --> Handlers

    %% Dependencias de Handlers
    AuthH --> TokStore
    AuthH --> UStore
    AuthH --> Crypto

    VehH --> VStore
    VehH --> UStore

    AdminH --> VStore
    AdminH --> UStore
    AdminH --> TokStore

    CatH --> VStore

    %% Dependencias de Seguridad y Dominio
    AuthMid --> TokStore
    AuditLogMid --> AuditFile

    UStore --> UserDom
    VStore --> VehDom
```

---

## 2. Diagrama B: Diagrama de Comportamiento (Secuencia AAA)

Ilustra la traza cronológica completa de una petición protegida (`POST /api/seller/vehicles`) atravesando el marco de seguridad AAA:
* **Accounting**: Inicia medición de latencia antes de procesar y registra la operación en `audit.log` al finalizar.
* **Authentication**: Valida la autenticidad y vigencia del token Bearer con `TokenStore`.
* **Authorization**: Verifica que el rol de la sesión sea `vendedor` mediante `RequireRole`.
* **Lógica y Almacenamiento**: Valida datos, asocia contacto del vendedor y persiste en estado `PENDING` de forma thread-safe.

```mermaid
sequenceDiagram
    autonumber
    actor Cliente as Vendedor / Cliente HTTP
    participant Gin as Gin Router
    participant Audit as AuditLogMiddleware (Accounting)
    participant Auth as AuthMiddleware (Authentication)
    participant TokS as TokenStore (Memoria)
    participant RBAC as RequireRole (Authorization)
    participant Handler as VehicleHandler
    participant UStore as UserStore
    participant VStore as VehicleStore
    participant AuditFile as audit.log (Disco)

    Cliente->>Gin: POST /api/seller/vehicles<br/>Header: Authorization: Bearer <token><br/>Body: {"title":"Mazda 3", "price":45000000, ...}
    
    activate Gin
    Gin->>Audit: Inicia pipeline de middlewares
    activate Audit
    Note over Audit: start = time.Now()<br/>Pasa el control hacia adelante vía c.Next()
    
    Audit->>Auth: Intercepta para autenticación
    activate Auth
    Auth->>TokS: ValidateToken(token)
    activate TokS
    TokS-->>Auth: Retorna Session{UserID, Role, ExpiresAt} (Válido)
    deactivate TokS
    Note over Auth: Inyecta sesión en contexto:<br/>c.Set("currentUser", session)
    
    Auth->>RBAC: Petición autenticada
    activate RBAC
    Note over RBAC: Verifica autorización RBAC:<br/>session.Role == "vendedor"
    
    RBAC->>Handler: Invoca CreatePublication(c)
    activate Handler
    Note over Handler: Valida DTO (TrimSpace, Price > 0, Year > 1900)
    
    Handler->>UStore: FindByID(session.UserID)
    activate UStore
    UStore-->>Handler: User{ContactInfo, ...}
    deactivate UStore
    
    Note over Handler: Construye VehiclePublication:<br/>• Status: PENDING (inmutable)<br/>• SellerID: session.UserID<br/>• SellerContact: user.ContactInfo
    
    Handler->>VStore: Create(pub)
    activate VStore
    Note over VStore: Lock() exclusivo -> Guarda en mapa -> Unlock()
    VStore-->>Handler: Confirmación
    deactivate VStore
    
    Handler-->>RBAC: HTTP 201 Created (JSON VehiclePublication)
    deactivate Handler
    RBAC-->>Auth: Propaga respuesta HTTP 201
    deactivate RBAC
    Auth-->>Audit: Retorna tras completar c.Next()
    deactivate Auth
    
    Note over Audit: duration = time.Since(start).Milliseconds()<br/>Extrae UserID y Role de la sesión<br/>Genera AuditLogEntry estructurado
    Audit->>AuditFile: Escribe línea JSON Lines [sync.Mutex]
    activate AuditFile
    AuditFile-->>Audit: Registro persistido en disco
    deactivate AuditFile
    
    Audit-->>Gin: Respuesta final consolidada
    deactivate Audit
    Gin-->>Cliente: HTTP 201 Created + JSON de la publicación
    deactivate Gin
```
