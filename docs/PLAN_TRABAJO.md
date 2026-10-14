# Plan de Trabajo y Backlog Scrum · AutoMarket Backend (AAA)

## 1. Resumen Ejecutivo del Proyecto y Reglas Clave

El objetivo es desarrollar un componente **Backend en Go** para **AutoMarket** implementando el marco de seguridad **AAA** (*Authentication, Authorization, Accounting*) y las Historias de Usuario para 3 roles: **Visitante**, **Vendedor** y **Administrador**.

### ⚠️ Reglas de Oro (Restricciones Críticas de Calidad y Evaluación)
1. **Tecnología Estricta:**
   - Lenguaje: **Go** (únicamente biblioteca estándar: `crypto/sha256`, `crypto/rand`, `crypto/hmac`, `os`, `encoding/json`, `sync`, etc.).
   - Única dependencia externa permitida: `github.com/gin-gonic/gin`.
   - **PROHIBIDO:** Librerías externas como `jwt-go`, `bcrypt`, `casbin`, `zap`, GORM, `pq`, `sqlite3`, etc.
2. **Persistencia:**
   - En memoria o archivos planos (`JSON`/archivos locales con `sync.RWMutex`) usando solo la stdlib.
3. **Cero Sobre-ingeniería (Alcance Estricto):**
   - El enunciado indica: *“Se penalizarán funcionalidades, endpoints, mecanismos o capacidades no solicitadas”*. No implementar recuperación de contraseña por email, OAuth, pasarelas de pago, etc.
4. **Entregables Finales:**
   - Documento PDF (portada alfabética Apellido-Nombre, repo commit hash, invite a `imcabezas`, link al video, 1 diagrama estructural, 1 diagrama de comportamiento, declaración IA).
   - Video de demostración/sustentación con Pull Request.

---

## 2. Definición del Marco AAA en Go (Solo Standard Library)

- **Authentication (Autenticación):** 
  - Hash de contraseñas: `crypto/sha256` o `crypto/sha512` con salt aleatorio (`crypto/rand`).
  - Tokens de sesión: Tokens opacos o tokens firmados tipo HMAC-SHA256 (`crypto/hmac`) codificados en hex/base64 generados con `crypto/rand`.
- **Authorization (Autorización):**
  - Middleware en Gin que extraiga el rol del usuario autenticado (`Visitante`, `Vendedor`, `Administrador`) y valide si el rol tiene permiso para invocar el endpoint solicitado.
- **Accounting (Trazabilidad / Auditoría):**
  - Middleware en Gin que registre de forma íntegra y cronológica cada operación: `timestamp`, `method`, `path`, `user_id`, `role`, `status_code`, `ip_address`, `latency`. Almacenamiento en archivo append-only (`audit.log`) o store thread-safe.

---

## 3. Asignación de Roles y Cargas (Equipo de 4 Integrantes)

Para lograr paralelismo sin pisarse código y con interfaces claras, la arquitectura se divide en 4 frentes:

| Integrante | Rol en el Proyecto                   | Responsabilidad Principal                                                                                                                                           |
| :--------- | :----------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Dev 1**  | **Tech Lead & Seguridad (AAA)**      | Núcleo de Seguridad AAA: Hash/Salt, Tokens de sesión, Middlewares de Autenticación, Autorización y Log de Auditoría (Accounting).                                   |
| **Dev 2**  | **Backend Core & Módulo Usuarios**   | Modelos base, Repositorio thread-safe de Usuarios, Endpoint de Registro/Login (`Vendedor`/`Admin`), Eliminación de usuarios (`Admin`).                              |
| **Dev 3**  | **Backend Catálogo & Publicaciones** | Módulo de Vehículos: Creación de publicación (`Vendedor`), Catálogo público con datos de contacto (`Visitante`), Reporte de venta (`Vendedor`).                     |
| **Dev 4**  | **Moderación, Integración & QA**     | Endpoints de Moderación (`Admin`: aprobar/eliminar publicación), Colección de pruebas (cURL/Postman), Diagramas (Estructural y Comportamiento) y Libreto del Video. |

---

## 4. Detalle de Responsabilidades por Integrante

### 👤 Integrante 1: Tech Lead & Seguridad (AAA)
- **Objetivo:** Garantizar el cumplimiento estricto de **RF-01**.
- **Entregables:**
  1. Utilidad criptográfica con stdlib (`hashPassword(pass, salt)`, `generateToken()`, `verifyPassword()`).
  2. Middleware de autenticación (`AuthMiddleware`): valida cabecera `Authorization: Bearer <token>` y monta usuario/rol en `c.Set()`.
  3. Middleware de autorización (`RequireRole(roles...)`): rechaza con 403 Forbidden si el rol no tiene permisos.
  4. Middleware de accounting (`AuditLogMiddleware`): intercepta request/response y escribe registros estructurados en log/almacén seguro.

### 👤 Integrante 2: Backend Core & Gestión de Usuarios
- **Objetivo:** Estructura base del proyecto y ciclo de vida de usuarios (**HU-VEN-01**, **HU-ADM-01**).
- **Entregables:**
  1. Inicialización de Go modules (`go mod init automarket`, instalación de `gin-gonic/gin`).
  2. Modelos de datos: `User` (`ID`, `Username`, `PasswordHash`, `Salt`, `Role`, `ContactInfo`).
  3. Almacenamiento en memoria/archivo con `sync.RWMutex` para usuarios.
  4. Endpoints:
     - `POST /api/auth/register` (Registro autónomo de vendedores con datos de contacto).
     - `POST /api/auth/login` (Autenticación y entrega de token).
     - `DELETE /api/admin/users/:id` (Eliminación de usuarios por Administrador).
  5. Precarga (seed) de un usuario `admin` inicial por configuración/arranque.

### 👤 Integrante 3: Módulo de Publicaciones y Catálogo
- **Objetivo:** Flujo de vehículos y catálogo público (**HU-VIS-01**, **HU-VEN-01**).
- **Entregables:**
  1. Modelo de datos: `VehiclePublication` (`ID`, `SellerID`, `Title`, `Description`, `Price`, `Year`, `Status: PENDING | APPROVED | SOLD`, `ContactInfo`).
  2. Almacenamiento en memoria/archivo con `sync.RWMutex` para publicaciones.
  3. Endpoints:
     - `GET /api/catalog` (Público - Visitante: lista vehículos en estado `APPROVED` con datos de contacto del vendedor).
     - `GET /api/catalog/:id` (Público - Visitante: detalle de publicación aprobada + contacto).
     - `POST /api/seller/vehicles` (Vendedor autenticado: publica vehículo en estado `PENDING`).
     - `PATCH /api/seller/vehicles/:id/sold` (Vendedor autenticado: reporta venta y retira del catálogo).

### 👤 Integrante 4: Moderación Admin, Pruebas y Documentación/Entrega
- **Objetivo:** Moderación de contenido (**HU-ADM-01**), integración global y preparación de entrega.
- **Entregables:**
  1. Endpoints de moderación para Administrador:
     - `GET /api/admin/publications/pending` (Listar pendientes de moderación).
     - `PATCH /api/admin/publications/:id/approve` (Aprobar publicación -> cambia a `APPROVED`).
     - `DELETE /api/admin/publications/:id` (Eliminar publicación).
  2. Script de pruebas / colección Postman o cURL demostrando todo el flujo de inicio a fin.
  3. Diagramas requeridos para el informe PDF:
     - **Diagrama de Estructura:** Diagrama de paquetes/componentes en Go (Clean Architecture / capas).
     - **Diagrama de Comportamiento:** Diagrama de secuencia del flujo AAA (p. ej., Vendedor envía publicación -> A
	 - **US-01 [Dev 2] Setup del Repositorio y Gin** `[ClickUp #86e3mc6x3]`
  - Inicializar `go mod init automarket` e instalar `github.com/gin-gonic/gin`.
  - Crear estructura: `cmd/server/`, `internal/domain/`, `internal/store/`, `internal/handlers/`, `internal/auth/`.
  - Crear `main.go` básico con endpoint `/ping` (200 OK) para probar el servidor.
  - Asegurar `.gitignore` para Go.
- **US-02 [Dev 2] Modelo y Repositorio de Usuarios** `[ClickUp #86e3mc6xk]`
  - Struct `User` en `internal/domain/user.go` (`ID`, `Username`, `PasswordHash`, `Salt`, `Role`, `ContactInfo`).
  - Almacén en memoria `UserStore` usando `map[string]User` protegido con `sync.RWMutex`.
  - Métodos: `Create`, `FindByUsername`, `FindByID`, `Delete`.
  - Función de arranque para crear un usuario inicial `admin`.
- **US-03 [Dev 1] Módulo Criptográfico Stdlib (Hash & Salt & Token)** `[ClickUp #86e3mc6y4]`
  - Generador de Salt seguro (16 bytes con `crypto/rand`).
  - Función `HashPassword(password, salt)` con `crypto/sha256`.
  - Función `CheckPassword(password, salt, hash) bool`.
  - Almacén de sesiones y generador de tokens aleatorios en memoria (`TokenStore`). *(Sin bcrypt ni JWT)*.
- **US-04 [Dev 1] Middlewares de Autenticación y Autorización** `[ClickUp #86e3mc6yu]`
  - Middleware `AuthMiddleware`: lee header `Authorization: Bearer <token>`, valida y monta usuario en `c.Set()`. Si falla, retorna `401 Unauthorized`.
  - Middleware `RequireRole(roles ...string)`: revisa el rol y retorna `403 Forbidden` si no coincide.
- **US-05 [Dev 2] Endpoints de Registro y Login** `[ClickUp #86e3mc6zd]`
  - `POST /api/auth/register`: valida datos, aplica hash/salt y registra vendedor con datos de contacto.
  - `POST /api/auth/login`: valida credenciales y entrega token de sesión.

---

### 📋 SPRINT 2: Funcionalidades de Negocio (HUs)
- **US-06 [Dev 3] Modelo y Repositorio de Publicaciones** `[ClickUp #86e3mc6zy]`
  - Struct `VehiclePublication` (`ID`, `SellerID`, `Title`, `Description`, `Price`, `Year`, `Status: PENDING | APPROVED | SOLD`, `ContactInfo`).
  - Almacén en memoria `PublicationStore` protegido con `sync.RWMutex`.
  - Métodos: `Create`, `FindByID`, `FindApproved`, `FindPending`, `UpdateStatus`, `Delete`.
- **US-07 [Dev 3] HU-VEN-01 - Crear Publicación de Vehículo** `[ClickUp #86e3mc70g]`
  - `POST /api/seller/vehicles` (protegido con `AuthMiddleware` + `RequireRole('vendedor')`).
  - Asocia `SellerID` del usuario autenticado y fuerza estado `PENDING`.
- **US-08 [Dev 4] HU-ADM-01 - Moderación de Publicaciones** `[ClickUp #86e3mc71b]`
  - `GET /api/admin/publications/pending`: listar pendientes de moderación.
  - `PATCH /api/admin/publications/:id/approve`: aprobar y cambiar estado a `APPROVED`.
  - `DELETE /api/admin/publications/:id`: eliminar publicación del sistema.
  - Proteger con `AuthMiddleware` + `RequireRole('admin')`.
- **US-09 [Dev 3] HU-VIS-01 - Catálogo Público de Vehículos** `[ClickUp #86e3mc71y]`
  - `GET /api/catalog`: lista pública de vehículos en estado `APPROVED` con datos de contacto del vendedor.
  - `GET /api/catalog/:id`: detalle de vehículo (404 si no existe o no está `APPROVED`).
  - Endpoints 100% públicos (sin autenticación).
- **US-10 [Dev 3] HU-VEN-01 - Reportar Vehículo Vendido** `[ClickUp #86e3mc72e]`
  - `PATCH /api/seller/vehicles/:id/sold` (protegido con rol vendedor).
  - Validar que quien ejecuta sea el dueño del vehículo (`SellerID == CurrentUser.ID`).
  - Cambiar estado a `SOLD` para retirarlo automáticamente del catálogo.
- **US-11 [Dev 2] HU-ADM-01 - Eliminación de Usuarios** `[ClickUp #86e3mc72v]`
  - `DELETE /api/admin/users/:id` (protegido con rol admin).
  - Elimina usuario del almacén e invalida sus sesiones.

---

### 📋 SPRINT 3: Accounting (Auditoría), QA, Diagramas y Video
- **US-12 [Dev 1] RF-01 Accounting - Middleware de Auditoría** `[ClickUp #86e3mc737]`
  - Struct `AuditLogEntry` (`timestamp`, `method`, `path`, `user_id`, `role`, `ip_address`, `status_code`, `duration_ms`).
  - Middleware global en Gin que escribe cada petición en `audit.log` de forma concurrente y segura.
- **US-13 [Dev 4] Script de Pruebas E2E (cURL/Postman)** `[ClickUp #86e3mc74b]`
  - Script secuencial demostrando: visitante (200), login/registro (200/201), vendedor publica (201 PENDING), admin aprueba (200), visitante ve en catálogo (200), vendedor marca vendido (200), intento sin token (401), intento sin rol (403).
  - Verificar que todo queda registrado en `audit.log`.
- **US-14 [Dev 4] Elaboración de Diagramas de Arquitectura** `[ClickUp #86e3mc754]`
  - 1 Diagrama Estructural: paquetes y componentes de Go.
  - 1 Diagrama de Comportamiento: secuencia HTTP a través de los middlewares AAA.
- **US-15 [Dev 1, 2, 3, 4] Grabación del Video de Sustentación** `[ClickUp #86e3mc75n]`
  - Video de sustenación y demostración asociado al Pull Request con participación de los 4 integrantes.
- **US-16 [Dev 4] Consolidación del Documento PDF Final** `[ClickUp #86e3mc767]`
  - Portada alfabética (Apellido - Nombre), commit hash, link a video, diagramas y declaración formal de IA.
- **US-17 [Dev 1] Invitación de GitHub a Docente** `[ClickUp #86e3mc76x]`
  - Enviar invitación de colaborador a `imcabezas` en el repositorio.

---

## 6. Convenciones de Trabajo en Git

1. **Ramas por tarea:** `feature/aaa-auth`, `feature/users-store`, `feature/vehicles-catalog`, `feature/admin-moderation`.
2. **Merge via Pull Request:** Todo código ingresa a `main` mediante PR revisado por al menos un compañero.
3. **Commits descriptivos:** `feat: implement user registration and password hashing`, `fix: enforce role check on admin delete`.
