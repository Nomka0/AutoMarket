# AutoMarket - Guión Explicativo del Proyecto Archivo por Archivo

Guión técnico y didáctico para exposición, defensa de código y auditoría técnica del proyecto **AutoMarket**. Explica el propósito, responsabilidades de diseño (Clean Architecture y Marco AAA), funciones, estructuras y líneas clave de cada archivo del repositorio.

---

## Índice General

1. [Punto de Entrada (`cmd/server/main.go`)](#1-punto-de-entrada-cmdservermaingo)
2. [Capa de Dominio (`internal/domain/`)](#2-capa-de-dominio-internaldomain)
   - [`user.go`](#21-internaldomainusergo)
   - [`vehicle.go`](#22-internaldomainvehiclego)
   - [`vehicle_test.go`](#23-internaldomainvehicle_testgo)
3. [Capa de Seguridad y Autenticación AAA (`internal/auth/`)](#3-capa-de-seguridad-y-autenticaci%C3%B3n-aaa-internalauth)
   - [`crypto.go`](#31-internalauthcryptogo)
   - [`crypto_test.go`](#32-internalauthcrypto_testgo)
   - [`token_store.go`](#33-internalauthtoken_storego)
   - [`middleware.go`](#34-internalauthmiddlewarego)
   - [`middleware_test.go`](#35-internalauthmiddleware_testgo)
   - [`audit.go`](#36-internalauthauditgo)
   - [`audit_test.go`](#37-internalauthaudit_testgo)
4. [Capa de Almacenamiento y Persistencia (`internal/store/`)](#4-capa-de-almacenamiento-y-persistencia-internalstore)
   - [`user_store.go`](#41-internalstoreuser_storego)
   - [`user_store_test.go`](#42-internalstoreuser_store_testgo)
   - [`vehicle_store.go`](#43-internalstorevehicle_storego)
   - [`vehicle_store_test.go`](#44-internalstorevehicle_store_testgo)
5. [Capa de Controladores y Handlers HTTP (`internal/handlers/`)](#5-capa-de-controladores-y-handlers-http-internalhandlers)
   - [`auth_handler.go`](#51-internalhandlersauth_handlergo)
   - [`auth_handler_test.go`](#52-internalhandlersauth_handler_testgo)
   - [`catalog_handler.go`](#53-internalhandlerscatalog_handlergo)
   - [`catalog_handler_test.go`](#54-internalhandlerscatalog_handler_testgo)
   - [`vehicle_handler.go`](#55-internalhandlersvehicle_handlergo)
   - [`vehicle_handler_test.go`](#56-internalhandlersvehicle_handler_testgo)
   - [`admin_handler.go`](#57-internalhandlersadmin_handlergo)
   - [`admin_handler_test.go`](#58-internalhandlersadmin_handler_testgo)
   - [`web_handler.go`](#59-internalhandlersweb_handlergo)
   - [`web_handler_test.go`](#510-internalhandlersweb_handler_testgo)
6. [Capa de Interfaz de Usuario (`templates/` y `static/`)](#6-capa-de-interfaz-de-usuario-templates-y-static)
   - [Plantillas HTML (`templates/*.html`)](#61-plantillas-html-templates)
   - [Hojas de Estilo (`static/style.css`)](#62-hoja-de-estilos-staticstylecss)
7. [Scripts de Automatización y Pruebas E2E (`test_flows.sh`)](#7-scripts-de-automatizaci%C3%B3n-y-pruebas-e2e-test_flowssh)
8. [Configuración y Dependencias](#8-configuraci%C3%B3n-y-dependencias)
   - [`go.mod` y `go.sum`](#81-gomod-y-gosum)
   - [`.env` y `.env.example`](#82-env-y-envexample)
   - [`.gitignore`](#83-gitignore)
9. [Documentación y Gestión del Proyecto (`docs/` y raíz)](#9-documentaci%C3%B3n-y-gesti%C3%B3n-del-proyecto-docs-y-ra%C3%ADz)

---

## 1. Punto de Entrada (`cmd/server/main.go`)

- **Ruta:** `cmd/server/main.go`
- **Propósito:** Orquestar el arranque del servidor, inyectar dependencias (Clean Architecture), registrar middlewares globales de auditoría y configurar el enrutamiento tanto para la API REST como para las vistas web SSR (Server-Side Rendering).

### Bloques de Código y Líneas Clave

- **Líneas 18–40 (`loadEnv(path string)`):**
  - Parser nativo de archivos de variables de entorno `.env` utilizando `bufio.NewScanner` y `os.Setenv`.
  - Ignora comentarios (`#`) y líneas vacías; divide cada línea en `KEY=VALUE` con `strings.SplitN(line, "=", 2)`.
  - *Decisión técnica:* Elimina la necesidad de librerías externas como `godotenv`, cumpliendo la restricción de usar stdlib pura de Go.
- **Líneas 43–55 (Inicialización de Entorno y Persistencia):**
  - Carga `.env`. Si `STORAGE_MODE == "memory"`, los paths de persistencia quedan vacíos para ejecutar en memoria volátil (ideal para pruebas).
  - Inicializa `store.NewUserStore("data/users.json")` y ejecuta `store.SeedAdmin`, sembrando el administrador inicial a partir de variables de entorno (`ADMIN_USERNAME` y `ADMIN_PASSWORD`).
- **Líneas 57–64 (Inicialización de Auditoría y Framework Gin):**
  - Instancia `auth.NewFileAuditLogger("audit.log")`.
  - Registra el middleware global `r.Use(auth.AuditLogMiddleware(auditLogger))` para garantizar que el 100% de las peticiones HTTP registren trazas de Accounting.
- **Líneas 65–67 (Configuración Web Estática y Plantillas):**
  - `r.Static("/static", "./static")` sirve CSS y recursos.
  - `r.LoadHTMLGlob("templates/*")` precompila las plantillas HTML para renderizado del lado del servidor.
- **Líneas 75–84 (Módulo de Autenticación):**
  - Instancia una única referencia compartida de `tokenStore := auth.NewTokenStore()`.
  - Registra rutas de autenticación en `/api/auth`: `/register`, `/login`, `/setup-admin`, `/register-admin` y `/has-admin`.
- **Líneas 85–98 (Módulos de Catálogo y Vendedor):**
  - Instancia `VehicleStore` y `VehicleHandler`.
  - `/api/catalog`: Público, sin middleware de autenticación (visitantes anónimos).
  - `/api/seller`: Protegido con `auth.AuthMiddleware(tokenStore)` y `auth.RequireRole(domain.RoleSeller)`.
- **Líneas 99–106 (Módulo de Administración):**
  - `/api/admin`: Protegido con `auth.AuthMiddleware(tokenStore)` y `auth.RequireRole(domain.RoleAdmin)`.
- **Líneas 108–134 (Rutas Web UI):**
  - Instancia `handlers.NewWebHandler` y registra rutas web para navegación con navegadores mediante cookies HttpOnly (`/login`, `/register`, `/seller`, `/admin`, `/catalog`).
- **Líneas 135–137:** Arranque del listener HTTP en `:8080`.

---

## 2. Capa de Dominio (`internal/domain/`)

### 2.1 `internal/domain/user.go`
- **Propósito:** Entidad pura del usuario y definición de roles del sistema.
- **Líneas Clave:**
  - **Líneas 3–6:** Constantes inmutables `RoleSeller = "vendedor"` y `RoleAdmin = "admin"`.
  - **Líneas 8–15 (`type User struct`):**
    - `ID string`: Identificador único de 64 caracteres hex.
    - `Username string`: Nombre de usuario único.
    - `PasswordHash string` y `Salt string`: Credenciales seguras (SHA-256 + Salt aleatorio).
    - `Role string`: Rol RBAC.
    - `ContactInfo string`: Correo o teléfono de contacto.
  - *Decisión de Ciberseguridad:* Las contraseñas en texto plano nunca se almacenan ni existen en el modelo. La protección contra fuga de credenciales se complementa con DTOs en los handlers.

### 2.2 `internal/domain/vehicle.go`
- **Propósito:** Entidad pura de la publicación de vehículo y máquina de estados de su ciclo de vida.
- **Líneas Clave:**
  - **Líneas 5–9:** Constantes de estado del vehículo:
    - `StatusPending = "PENDING"` (recién creada por vendedor, oculta al público).
    - `StatusApproved = "APPROVED"` (aprobada por admin, visible en catálogo).
    - `StatusSold = "SOLD"` (marcada por el vendedor dueño, retirada de la venta).
  - **Líneas 11–21 (`type VehiclePublication struct`):**
    - `SellerID string`: Dueño del vehículo (soporte para control de acceso horizontal).
    - `SellerContact string`: Datos de contacto expuestos al visitante en catálogo.
    - `Title`, `Description`, `Price`, `Year`, `Status`, `CreatedAt`.
  - **Líneas 23–25 (`IsValidStatus(status string) bool`):**
    - Valida que un estado pertenezca a los tres valores autorizados.

### 2.3 `internal/domain/vehicle_test.go`
- **Propósito:** Pruebas unitarias de las reglas de dominio del vehículo.
- **Líneas Clave:**
  - Comprueba que `IsValidStatus` retorne `true` para `PENDING`, `APPROVED` y `SOLD`, y `false` ante cadenas arbitrarias o vacías.

---

## 3. Capa de Seguridad y Autenticación AAA (`internal/auth/`)

Implementa el marco **AAA** (*Authentication, Authorization, Accounting*) sin bibliotecas externas (prohibido JWT y bcrypt según el enunciado).

### 3.1 `internal/auth/crypto.go`
- **Propósito:** Primitivas criptográficas estándar para salting, hashing y generación de tokens de sesión opacos.
- **Líneas Clave:**
  - **Líneas 9–13 (`GenerateSalt() string`):**
    - Lee 16 bytes pseudoaleatorios criptográficos con `crypto/rand.Read` y los codifica en hexadecimal (32 caracteres).
    - *Seguridad:* Garantiza un salt único por usuario, mitigando ataques de tablas Rainbow y ataques de diccionario masivos.
  - **Líneas 15–18 (`HashPassword(password, salt string) string`):**
    - Computa `sha256.Sum256([]byte(password + salt))` y devuelve 64 caracteres hexadecimales.
  - **Líneas 20–26 (`CheckPassword` / `VerifyPassword`):**
    - Verifica si el hash de la contraseña provista junto con el salt coincide con el hash almacenado.
  - **Líneas 28–32 (`GenerateToken() string`):**
    - Genera 32 bytes de entropía con `crypto/rand` (64 caracteres hex).
    - Produce **tokens opacos** imposibles de falsificar o descifrar sin acceso al estado del servidor.

### 3.2 `internal/auth/crypto_test.go`
- **Propósito:** Pruebas unitarias de determinismo criptográfico, unicidad de salts/tokens y verificación de contraseñas.
- **Líneas Clave:**
  - Valida longitudes exactas (salt 32 chars, token 64 chars), unicidad entre llamadas y rechazo ante contraseñas inválidas.
  - Prueba concurrencia con múltiples goroutines creando y consultando tokens simultáneamente (`TestTokenStoreConcurrency`).

### 3.3 `internal/auth/token_store.go`
- **Propósito:** Gestión concurrente de sesiones activas en memoria volátil (Authentication - A1).
- **Líneas Clave:**
  - **Línea 8:** `sessionTTL = 24 * time.Hour`: Tiempo de vida de sesión.
  - **Líneas 10–19 (`Session` y `TokenStore`):**
    - `Session`: Guarda `UserID`, `Role` y `ExpiresAt`.
    - `TokenStore`: Diccionario protegido por `sync.RWMutex`.
  - **Líneas 27–37 (`CreateToken` y `Set`):** Asigna un token a una sesión con expiración a 24 horas.
  - **Líneas 40–48 (`ValidateToken(token string) (Session, bool)`):**
    - Bloqueo de lectura `mu.RLock()`.
    - Valida que el token exista y que `time.Now().Before(session.ExpiresAt)`. Retorna sesión vacía y `false` si expiró.
  - **Líneas 61–69 (`DeleteByUserID(userID string)`):**
    - Invalida inmediatamente todos los tokens asociados a un ID de usuario.
    - *Ciberseguridad:* Utilizado cuando un administrador elimina una cuenta; revoca sesiones activas al instante.
  - *Decisión de Arquitectura:* Los tokens no se persisten en disco; residen en RAM para que al reiniciar el servidor queden invalidadas las sesiones huérfanas y se reduzca la superficie de ataque ante robo de archivos de datos.

### 3.4 `internal/auth/middleware.go`
- **Propósito:** Interceptores de solicitud HTTP para autenticación y autorización (Authorization - A2).
- **Líneas Clave:**
  - **Líneas 13–30 (`AuthMiddleware(store *TokenStore)`):**
    - Extrae el encabezado `Authorization: Bearer <token>`.
    - Si no existe el prefijo o el token es inválido/expirado, detiene la cadena con `c.AbortWithStatusJSON(401, ...)`.
    - Si es válido, inyecta la sesión en el contexto Gin: `c.Set("currentUser", session)`.
  - **Líneas 32–45 (`RequireRole(allowedRoles ...string)`):**
    - Control de Acceso Basado en Roles (RBAC).
    - Obtiene la sesión con `GetCurrentUser(c)`. Si el rol no figura en `allowedRoles`, responde `403 Forbidden`.
  - **Líneas 56–75 (`WebAuthMiddleware`) & 88–107 (`WebRequireRole`):**
    - Variantes para la interfaz web basadas en cookie `token` (HttpOnly). Redirigen a `/login` o muestran `error.html` con status 403.
  - **Líneas 77–86 (`WebOptionalAuthMiddleware`):**
    - Permite a visitantes anónimos navegar sin sesión, pero si tienen cookie válida, inyecta sus datos para personalizar el encabezado (mostrar nombre y enlaces correspondientes).

### 3.5 `internal/auth/middleware_test.go`
- **Propósito:** Pruebas de integración de middlewares simulando requests HTTP con `httptest.NewRecorder`.
- **Líneas Clave:**
  - Verifica casos de éxito (200), ausencia de header (401), formato mal formado (401), token vencido (401) y rol denegado (403).

### 3.6 `internal/auth/audit.go`
- **Propósito:** Trazabilidad inmutable de todas las acciones del sistema (Accounting - A3).
- **Líneas Clave:**
  - **Líneas 13–22 (`type AuditLogEntry struct`):**
    - Estructura JSON con: `timestamp` (UTC RFC3339), `method`, `path`, `ip`, `user_id` (o `"anonymous"`), `role` (o `"visitor"`), `status_code` y `duration_ms`.
  - **Líneas 24–45 (`AuditLogger` y `NewAuditLogger`):**
    - Maneja la escritura concurrente en archivo `audit.log` con `os.O_APPEND|os.O_CREATE|os.O_WRONLY` y `sync.Mutex`.
  - **Líneas 55–64 (`Log(entry AuditLogEntry)`):**
    - Serializa la entrada a JSON y escribe una línea por evento (JSON Lines format).
  - **Líneas 67–101 (`AuditLogMiddleware(logger *AuditLogger)`):**
    - Intercepta la petición, mide `time.Since(start).Milliseconds()`, extrae el usuario autenticado (si lo hay) y registra el evento con el código de respuesta real (`c.Writer.Status()`).

### 3.7 `internal/auth/audit_test.go`
- **Propósito:** Pruebas del registro de auditoría y formato JSON Lines en disco.
- **Líneas Clave:**
  - Genera entradas, lee el archivo temporal y valida que los campos obligatorios contengan valores correctos.

---

## 4. Capa de Almacenamiento y Persistencia (`internal/store/`)

Implementa repositorios concurrentes desacoplados con sincronización a archivos JSON locales mediante stdlib pura (`encoding/json`, `os`, `sync.RWMutex`).

### 4.1 `internal/store/user_store.go`
- **Propósito:** Repositorio de usuarios con persistencia en `data/users.json`.
- **Líneas Clave:**
  - **Líneas 14–18 (`type UserStore struct`):**
    - `mu sync.RWMutex`: Soporta múltiples lectores simultáneos y exclusión mutua para escrituras.
    - `users map[string]domain.User`: Clave indexada por ID de usuario.
    - `filePath string`: Ruta del archivo JSON de persistencia.
  - **Líneas 31–61 (`load()` y `save()`):**
    - `load`: Carga el archivo en el mapa en memoria. Si no existe, no falla (arranque en frío).
    - `save`: Crea directorios si no existen (`os.MkdirAll`) y escribe atómicamente con `os.WriteFile`.
  - **Líneas 63–76 (`Create(u domain.User) error`):**
    - Verifica unicidad de nombre de usuario (`username`). Si ya existe, retorna error.
  - **Líneas 78–96 (`FindByUsername` y `FindByID`):**
    - Búsqueda segura con bloqueo de lectura `s.mu.RLock()`.
  - **Líneas 98–108 (`Delete(id string) bool`):**
    - Elimina al usuario del mapa y sincroniza a disco.
  - **Líneas 120–130 (`HasAdmin() bool`):**
    - Comprueba si ya existe un usuario con `RoleAdmin`.
  - **Líneas 132–158 (`CreateAdmin(username, password, contactInfo string)`):**
    - Impide la creación de más de un administrador (`"admin user already exists"`).
    - Genera salt, hash y asigna `RoleAdmin`.
  - **Líneas 160–177 (`SeedAdmin`):**
    - Si existen las variables `ADMIN_USERNAME` y `ADMIN_PASSWORD` y el sistema no tiene administrador, lo aprovisiona automáticamente al inicio.

### 4.2 `internal/store/user_store_test.go`
- **Propósito:** Pruebas unitarias de persistencia, unicidad de usuarios y aprovisionamiento de admin.
- **Líneas Clave:**
  - Evalúa creación, búsqueda, bloqueo de usuarios duplicados, unicidad de admin y persistencia/recarga desde disco.

### 4.3 `internal/store/vehicle_store.go`
- **Propósito:** Repositorio de publicaciones de vehículos con persistencia en `data/vehicles.json`.
- **Líneas Clave:**
  - **Líneas 12–16 (`type VehicleStore struct`):**
    - Mapa `pubs map[string]domain.VehiclePublication` protegido por `sync.RWMutex`.
  - **Líneas 61–66 (`Create(pub domain.VehiclePublication)`):**
    - Inserta la publicación y la persiste en disco.
  - **Líneas 75–85 (`FindApproved() []domain.VehiclePublication`):**
    - Filtra y retorna únicamente vehículos en estado `APPROVED`.
    - Inicializa `res := []domain.VehiclePublication{}` para garantizar que en JSON serialice como array vacío `[]` y nunca como `null`.
  - **Líneas 87–97 (`FindPending()`):**
    - Retorna vehículos en estado `PENDING` para la cola de moderación del administrador.
  - **Líneas 99–109 (`FindBySellerID(sellerID string)`):**
    - Retorna todos los vehículos de un vendedor específico para su panel de control.
  - **Líneas 111–122 (`UpdateStatus(id, newStatus string) bool`):**
    - Actualiza el estado de la publicación bajo candado de escritura `mu.Lock()`.
  - **Líneas 124–133 (`Delete(id string) bool`):**
    - Elimina la publicación y guarda cambios.

### 4.4 `internal/store/vehicle_store_test.go`
- **Propósito:** Pruebas de filtrado por estados, consultas por vendedor y operaciones atómicas de actualización/borrado.

---

## 5. Capa de Controladores y Handlers HTTP (`internal/handlers/`)

Traduce solicitudes HTTP, ejecuta validaciones de entrada, aplica saneamiento de datos y coordina casos de uso con las capas de dominio, seguridad y persistencia.

### 5.1 `internal/handlers/auth_handler.go`
- **Propósito:** Endpoints REST de autenticación y registro de usuarios (`/api/auth`).
- **Líneas Clave:**
  - **Líneas 13–22:** DTOs de entrada: `RegisterRequest` y `LoginRequest` con tags `binding:"required"`.
  - **Líneas 33–49 (`Login(c *gin.Context)`):**
    - Valida credenciales contra `userStore` y `auth.CheckPassword`.
    - *Protección contra enumeración de usuarios:* Devuelve el mismo código `401 Unauthorized` (`"invalid credentials"`) tanto si el usuario no existe como si la contraseña es incorrecta.
    - Si es válido, emite un token de sesión opaco con `tokenStore.CreateToken(user.ID, user.Role)`.
  - **Líneas 51–74 (`Register(c *gin.Context)`):**
    - Asigna forzosamente `Role: domain.RoleSeller` (L64). El registro público jamás permite autoasignarse rol de administrador.
    - Genera salt criptográfico y hash SHA-256 antes de persistir.
  - **Líneas 76–105 (`SetupAdmin(c *gin.Context)`):**
    - Permite registrar el primer administrador. Si ya existe uno, responde `403 Forbidden` (`"admin user already exists"`).
  - **Líneas 107–109 (`HasAdmin(c *gin.Context)`):**
    - Informa si el sistema ya cuenta con administrador configurado.

### 5.2 `internal/handlers/auth_handler_test.go`
- **Propósito:** Pruebas de integración sobre endpoints de login, registro público y setup de administrador.

### 5.3 `internal/handlers/catalog_handler.go`
- **Propósito:** Endpoints públicos para visitantes anónimos (`/api/catalog`, HU-VIS-01).
- **Líneas Clave:**
  - **Líneas 20–24 (`GetCatalog`):**
    - Llama a `h.vehicleStore.FindApproved()`.
    - *Seguridad:* Los vehículos en estado `PENDING` o `SOLD` se filtran en la capa de persistencia; nunca llegan al cliente público.
  - **Líneas 26–35 (`GetCatalogDetail`):**
    - Consulta un vehículo por ID.
    - Si no existe o su estado no es `APPROVED`, devuelve `404 Not Found`. Un visitante anónimo no puede determinar si un ID existe si está pendiente o vendido.

### 5.4 `internal/handlers/catalog_handler_test.go`
- **Propósito:** Valida que el catálogo público solo exponga publicaciones aprobadas y oculte pendientes/vendidas.

### 5.5 `internal/handlers/vehicle_handler.go`
- **Propósito:** Endpoints para gestión de publicaciones por parte del vendedor (`/api/seller/vehicles`, HU-VEN-01).
- **Líneas Clave:**
  - **Líneas 15–20 (`CreatePublicationRequest`):** DTO con validaciones (`gt=0` en precio, `gt=1900` en año).
  - **Líneas 31–61 (`CreatePublication`):**
    - Sanea títulos y descripciones con `strings.TrimSpace` para rechazar entradas compuestas exclusivamente por espacios en blanco (L34).
    - Asocia el `SellerID` obtenido de la sesión autenticada (`session.UserID`).
    - Asigna inmutablemente el estado inicial `Status: domain.StatusPending` (L55). Ninguna publicación de vendedor entra directamente aprobada al catálogo.
  - **Líneas 63–91 (`MarkAsSold`):**
    - *Control de Acceso Horizontal (Ownership):* Comprueba en L76 que `pub.SellerID == session.UserID`. Si un vendedor intenta marcar como vendido un vehículo ajeno, responde `403 Forbidden`.
    - *Validación de Transición de Estado:* Verifica en L81 que `pub.Status == domain.StatusApproved`. No permite marcar como vendida una publicación pendiente ni una ya vendida (responde `409 Conflict`).

### 5.6 `internal/handlers/vehicle_handler_test.go`
- **Propósito:** Pruebas completas de creación en estado pendiente, validaciones de saneamiento y control horizontal de propiedad.

### 5.7 `internal/handlers/admin_handler.go`
- **Propósito:** Endpoints de moderación y administración (`/api/admin`, HU-ADM-01).
- **Líneas Clave:**
  - **Líneas 27–30 (`GetPendingPublications`):**
    - Retorna publicaciones en estado `PENDING` para revisión del administrador.
  - **Líneas 32–51 (`ApprovePublication`):**
    - Transición de estado `PENDING -> APPROVED`.
    - Si el vehículo ya estaba aprobado o vendido, retorna `409 Conflict` (L41).
  - **Líneas 53–61 (`DeletePublication`):**
    - Eliminación atómica de publicaciones rechazadas o inadecuadas.
  - **Líneas 63–84 (`DeleteUser`):**
    - *Regla de Seguridad:* Impide que el administrador se autoelimine (`session.UserID == id -> 400 Bad Request`, L67).
    - Elimina al usuario de `UserStore` y revoca en cascada todas sus sesiones activas en `TokenStore` (L79–81).

### 5.8 `internal/handlers/admin_handler_test.go`
- **Propósito:** Pruebas de moderación, bloqueos ante dobles aprobaciones y revocación de tokens al eliminar usuarios.

### 5.9 `internal/handlers/web_handler.go`
- **Propósito:** Controlador Web completo con Server-Side Rendering (SSR) y autenticación mediante cookies seguras.
- **Líneas Clave:**
  - **Líneas 36–67 (`Catalog` y `CatalogDetail`):** Renderiza vistas públicas con información contextual del usuario autenticado (si lo hay).
  - **Líneas 69–106 (`LoginPage` y `LoginSubmit`):** Procesa formulario de login web y establece la cookie `token` (`HttpOnly: true`, duración 24 horas).
  - **Líneas 108–114 (`Logout`):** Invalida el token en `TokenStore` y expira la cookie.
  - **Líneas 116–152 (`RegisterPage` y `RegisterSubmit`):** Procesa registro de vendedores web.
  - **Líneas 154–188 (`SetupAdminPage` y `SetupAdminSubmit`):** Interfaz para aprovisionar el primer admin.
  - **Líneas 190–262 (`SellerDashboard` y `CreateVehicleSubmit`):**
    - Dashboard con contadores de publicaciones (`PendingCount`, `ApprovedCount`, `SoldCount`).
    - Valida precio y año de vehículo (`1900 <= year <= añoActual + 1`).
  - **Líneas 264–286 (`MarkSoldSubmit`):** Valida propiedad y cambia estado a `SOLD` desde la interfaz web.
  - **Líneas 288–342 (`AdminDashboard`, `ApprovePublicationSubmit`, `DeletePublicationSubmit`, `DeleteUserSubmit`):**
    - Panel de moderación de publicaciones y gestión de cuentas de usuario.

### 5.10 `internal/handlers/web_handler_test.go`
- **Propósito:** Pruebas de endpoints web, redirecciones HTTP (302/303), seteo de cookies y renderizado de plantillas HTML.

---

## 6. Capa de Interfaz de Usuario (`templates/` y `static/`)

### 6.1 Plantillas HTML (`templates/`)
Todas las plantillas utilizan el motor estándar `html/template` de Go, que incluye escape automático contextual contra vulnerabilidades XSS (*Cross-Site Scripting*).

- **`templates/catalog.html`:** Vista pública del catálogo. Muestra cards de vehículos aprobados, precios formateados y badges de estado para visitantes y usuarios logueados.
- **`templates/catalog_detail.html`:** Vista de detalle del vehículo. Muestra especificaciones completas y datos de contacto directo del vendedor (`seller_contact`).
- **`templates/login.html`:** Formulario de inicio de sesión con alertas de error y enlace dinámico a `setup-admin` si el sistema no tiene administrador.
- **`templates/register.html`:** Formulario de registro de nuevos vendedores con campos obligatorios de contacto.
- **`templates/setup_admin.html`:** Pantalla de configuración inicial del administrador (solo accesible si no existe admin).
- **`templates/seller.html`:** Panel del vendedor con métricas en tiempo real, formulario de publicación y tabla de vehículos propios con botón para reportar como vendido.
- **`templates/admin.html`:** Panel del moderador con listado de publicaciones pendientes (aprobar/eliminar) y gestión de usuarios (eliminar vendedor).
- **`templates/error.html`:** Página de error amigable para accesos denegados (403) o recursos no encontrados (404).

### 6.2 Hoja de Estilos (`static/style.css`)
- **Propósito:** Diseño CSS puro sin frameworks pesados, con paleta sobria, componentes modulares (cards, badges, alertas, tablas, formularios) y adaptabilidad responsiva para escritorio y móvil.

---

## 7. Scripts de Automatización y Pruebas E2E (`test_flows.sh`)

- **Ruta:** `test_flows.sh`
- **Propósito:** Batería completa de pruebas funcionales y de seguridad de extremo a extremo (E2E) sobre la API HTTP en vivo.
- **Estructura y Pasos de Verificación:**
  - **Paso 0 (L46–55):** Comprueba conectividad contra `/ping`.
  - **Paso 1 (L64–67):** Verifica que el catálogo público sea accesible sin credenciales (HU-VIS-01).
  - **Paso 2 (L69–109):** Setup de administrador inicial, rechazo con 403 a un segundo administrador, registro de dos vendedores y rechazo de nombres duplicados.
  - **Paso 3 (L111–144):** Inicio de sesión de vendedores y admin, emisión de tokens Bearer y rechazo de contraseñas erróneas con 401.
  - **Paso 4 (L146–182):** Intento de publicación sin token (401), intento con rol admin (403), rechazo de campos con solo espacios (400) y publicación exitosa en estado `PENDING`.
  - **Paso 5 (L184–233):** Moderación admin: listado de pendientes (401 sin token, 403 para vendedor, 200 para admin); confirmación de que la publicación pendiente no aparece en catálogo; aprobación (`APPROVED`) y rechazo a doble aprobación (409).
  - **Paso 6 (L235–254):** Verificación de que la publicación aprobada ahora es visible en catálogo y expone los datos de contacto del vendedor.
  - **Paso 7 (L256–283):** Control de acceso horizontal: un vendedor no dueño intenta marcar como vendido y es rechazado con 403; el dueño legítimo lo marca con 200; el vehículo se retira inmediatamente del catálogo público.
  - **Paso 8 (L285–302):** Eliminación de usuarios: admin no puede autoeliminarse (400); eliminación de vendedor (200) e invalidación automática de su token previo (401).
  - **Paso 9 (L304–326):** Verificación del archivo `audit.log`: valida existencia física, acumulación de registros y cumplimiento de los campos obligatorios del marco AAA (Accounting).

---

## 8. Configuración y Dependencias

### 8.1 `go.mod` y `go.sum`
- **Módulo:** `automarket`
- **Versión de Go:** `1.25.0`
- **Dependencia Directa Única:** `github.com/gin-gonic/gin v1.12.0` (framework web de alto rendimiento autorizado para el taller). Todas las demás capacidades (criptografía, concurrencia, persistencia JSON y plantillas) provienen de la biblioteca estándar de Go.

### 8.2 `.env` y `.env.example`
- **Propósito:** Configuración de arranque del servidor.
- **Variables:**
  - `STORAGE_MODE`: `"file"` (persistencia JSON en disco) o `"memory"` (memoria volátil).
  - `ADMIN_USERNAME`: Nombre de usuario del administrador inicial para el seeding automático.
  - `ADMIN_PASSWORD`: Contraseña del administrador inicial.

### 8.3 `.gitignore`
- **Propósito:** Prevenir la inclusión en el repositorio Git de binarios compilados, ejecutables locales, variables sensibles (`.env`), registros de auditoría (`audit.log`) y datos locales (`data/*.json`).

---

## 9. Documentación y Gestión del Proyecto (`docs/` y raíz)

- **`README.md`:** Guía rápida de instalación, ejecución del servidor, credenciales por defecto y comandos de prueba.
- **`docs/enunciado.md`:** Enunciado oficial del taller académico con requisitos funcionales, historias de usuario y restricciones de ciberseguridad.
- **`docs/PLAN_TRABAJO.md`:** Plan de trabajo detallado con estimación de horas, fases de desarrollo y asignación de tareas.
- **`docs/DIAGRAMAS_ARQUITECTURA.md`:** Diagramas Mermaid de arquitectura en capas, flujo de autenticación AAA y ciclo de vida de los vehículos.
- **`docs/GUIA_ARQUITECTURA_Y_SUSTENTACION.md`:** Documento de sustento teórico y técnico con justificaciones arquitectónicas y banco de preguntas de examen.
- **`docs/REQUERIMIENTOS_ENTREGA_PDF.md`:** Matriz de trazabilidad y checklist de cumplimiento de los requisitos de entrega.
- **`docs/clickup_tasks.csv`:** Registro estructurado de tareas, estados e historias de usuario para seguimiento en ClickUp.
