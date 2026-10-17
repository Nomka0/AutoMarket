# Guión de Sustentación Técnica: AutoMarket Backend
## Implementación Integral del Marco de Seguridad AAA (*Authentication, Authorization, Accounting*)

* **Duración estimada:** 13 - 14 minutos (Límite máximo permitido: 15 minutos).
* **Enfoque:** Arquitectura Backend Clean Architecture (Go stdlib pura + Gin), Ciberseguridad Defensiva y Demostración en la Interfaz Web.
* **Estructura del Video:**
  1. **Apertura y Arquitectura General** (00:00 - 01:45)
  2. **Pilar 1: Authentication (Autenticación)** (01:45 - 05:45)
  3. **Pilar 2: Authorization (Autorización)** (05:45 - 10:15)
  4. **Pilar 3: Accounting (Auditoría / Trazabilidad)** (10:15 - 12:30)
  5. **Verificación Automatizada E2E y Cierre** (12:30 - 14:15)

---

## 1. Apertura y Arquitectura de la Solución (00:00 - 01:45)

### [PANTALLA / ACCIÓN]
* Mostrar en el editor la estructura de directorios del proyecto (`cmd/server`, `internal/domain`, `internal/store`, `internal/auth`, `internal/handlers`, `templates/`).
* Abrir brevemente el archivo [`go.mod`](file:///home/nomka/Projects/AutoMarket/go.mod) para evidenciar las dependencias.
* Mostrar en navegador la página principal [`http://localhost:8080/catalog`](http://localhost:8080/catalog).

### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “Cordial saludo. En este video realizaremos la sustentación técnica del componente backend y la plataforma web de **AutoMarket**, desarrollada para el taller de ciberseguridad sobre el **Marco de Seguridad AAA** (*Authentication, Authorization, Accounting*).
> 
> **Cumplimiento estricto de restricciones técnicas:**
> Como se observa en nuestro [`go.mod`](file:///home/nomka/Projects/AutoMarket/go.mod), la solución cumple rigurosamente la regla del proyecto: empleamos únicamente la biblioteca estándar de Go y `github.com/gin-gonic/gin` como única dependencia externa. No utilizamos librerías de terceros prohibidas como JWT, bcrypt, Casbin, ORMs ni drivers externos de bases de datos.
> 
> **Diseño bajo Clean Architecture:**
> Organizamos el sistema en capas desacopladas con responsabilidades unívocas:
> 1. `internal/domain`: Entidades puras y reglas de negocio (`User`, `VehiclePublication`).
> 2. `internal/store`: Repositorios concurrentes con persistencia híbrida en memoria RAM sincronizada a archivos JSON (`users.json` y `vehicles.json`), protegidos con cerrojos `sync.RWMutex`.
> 3. `internal/auth`: Primitivas criptográficas nativas, almacén de sesiones volátiles y middlewares de seguridad.
> 4. `internal/handlers`: Controladores HTTP para la API REST y vistas web renderizadas con `html/template`.
> 
> A continuación, desglosaremos la solución organizándola estrictamente bajo los **3 Pilares del Marco AAA**, conectando cada decisión de arquitectura backend con su comportamiento en la interfaz de usuario.”

---

## 2. Pilar 1: Authentication (Autenticación) (01:45 - 05:45)

### A. Fundamento Criptográfico y Modelo de Sesiones en Backend

#### [PANTALLA / ACCIÓN]
* Abrir el código de [`internal/auth/crypto.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/crypto.go#L9-L32) destacando las funciones `GenerateSalt`, `HashPassword`, `CheckPassword` y `GenerateToken`.
* Abrir [`internal/auth/token_store.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/token_store.go#L10-L48) destacando la estructura `Session` y la expiración con TTL de 24 horas.

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “El primer pilar, **Authentication**, garantiza que el sistema valide fehacientemente la identidad de cualquier entidad antes de permitirle operar sobre recursos protegidos.
> 
> **Criptografía nativa sin bcrypt:**
> En [`internal/auth/crypto.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/crypto.go):
> * `GenerateSalt()`: Extrae 16 bytes de entropía criptográfica real del sistema operativo mediante `crypto/rand.Read` y los codifica en una cadena hexadecimal de 32 caracteres. Esto asegura que cada cuenta tenga una sal única, neutralizando ataques de diccionario y tablas Rainbow precalculadas.
> * `HashPassword()` y `CheckPassword()`: Concatenan la contraseña en texto plano con el salt del usuario y generan el resumen determinista de 64 caracteres hexadecimales con `crypto/sha256`.
> 
> **Tokens Opacos en RAM vs JWT:**
> Ante la restricción de no usar JWT y por buenas prácticas de ciberseguridad:
> * `GenerateToken()` produce cadenas criptográficamente aleatorias de 32 bytes (64 caracteres hexadecimales). A diferencia de un JWT, que serializa datos del usuario visibles para el cliente, nuestros tokens son **completamente opacos**: son punteros aleatorios a una sesión que solo existe en la memoria del servidor.
> * En [`internal/auth/token_store.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/token_store.go), protegemos las sesiones con un `sync.RWMutex`. Cada sesión almacena `UserID`, `Role` y un TTL de expiración de 24 horas (`ExpiresAt`).
> * **Decisión de seguridad en reposo:** Los tokens residen exclusivamente en memoria volátil y jamás se persisten en disco. Si los archivos locales de datos sufrieran una fuga o robo, ningún atacante podría extraer sesiones activas (*session hijacking*).”

---

### B. Controles Defensivos en Handlers y Demostración en la Interfaz Web

#### [PANTALLA / ACCIÓN]
* Mostrar en código [`internal/handlers/auth_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/auth_handler.go#L33-L74) y [`internal/handlers/web_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/web_handler.go#L89-L153).
* Cambiar al navegador web en [`http://localhost:8080/setup-admin`](http://localhost:8080/setup-admin).
* Demostrar el intento de ingresar a `/setup-admin` (o registrar segundo admin) y ver el rechazo.
* Ir a [`http://localhost:8080/register`](http://localhost:8080/register): registrar un vendedor (ej. `vendedor_juan`).
* Abrir las **DevTools del navegador** (pestaña *Application* / *Almacenamiento* -> *Cookies*).
* Mostrar la cookie de nombre `token`: señalar que tiene activada la bandera **`HttpOnly`**.
* Hacer clic en **Cerrar Sesión** (`/logout`) y mostrar cómo la cookie se purga inmediatamente.
* Intentar iniciar sesión con contraseña errónea para evidenciar el mensaje genérico.

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “Conectemos ahora la arquitectura de autenticación con la interfaz de usuario:
> 
> **1. Aprovisionamiento seguro del Administrador:**
> Durante el arranque del servidor en `cmd/server/main.go`, `store.SeedAdmin` lee credenciales del archivo `.env` mediante nuestro parser nativo `loadEnv` y crea al administrador si no existe.
> Si un usuario navega a [`/setup-admin`](file:///home/nomka/Projects/AutoMarket/internal/handlers/web_handler.go#L155) o invoca `POST /api/auth/setup-admin`, el método `HasAdmin()` del repositorio detecta que ya existe una cuenta de administrador y corta la operación con `403 Forbidden`, impidiendo que un atacante suplante el control de la plataforma.
> 
> **2. Mitigación de Mass Assignment en el Registro:**
> En [`internal/handlers/auth_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/auth_handler.go#L64) y en la vista [`/register`](file:///home/nomka/Projects/AutoMarket/internal/handlers/web_handler.go#L141), el backend fuerza de manera inmutable el rol `domain.RoleSeller`. Aunque un usuario modifique el payload HTTP enviando `"role": "admin"`, el controlador ignora dicha entrada y asigna estrictamente el rol de vendedor.
> 
> **3. Mitigación de Enumeración de Usuarios en el Login:**
> Si un usuario ingresa credenciales falsas en [`/login`](file:///home/nomka/Projects/AutoMarket/internal/handlers/web_handler.go#L94), el backend responde con el mensaje genérico uniforme `'Credenciales inválidas'` (código `401 Unauthorized`), impidiendo que un atacante descubra qué nombres de usuario existen en la base de datos.
> 
> **4. Protección de Sesión Web mediante Cookies HttpOnly:**
> Al autenticarse exitosamente, el backend emite el token opaco y lo adjunta con `c.SetCookie("token", token, 86400, "/", "", false, true)`. Como observamos en las herramientas de desarrollador, el atributo **HttpOnly** está habilitado: esto neutraliza el robo de sesión mediante ataques de *Cross-Site Scripting* (XSS), ya que ningún script malicioso ejecutado en el navegador puede acceder a la cookie.”

---

## 3. Pilar 2: Authorization (Autorización) (05:45 - 10:15)

### A. Control de Acceso RBAC, Propiedad Horizontal y Ciclo de Vida en Backend

#### [PANTALLA / ACCIÓN]
* Mostrar en código [`internal/auth/middleware.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/middleware.go#L13-L107) resaltando `RequireRole` y `WebRequireRole`.
* Mostrar en [`internal/domain/vehicle.go`](file:///home/nomka/Projects/AutoMarket/internal/domain/vehicle.go#L5-L25) los estados `StatusPending`, `StatusApproved` y `StatusSold`.
* Mostrar en [`internal/handlers/vehicle_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/vehicle_handler.go#L74-L85) la validación horizontal `pub.SellerID == session.UserID`.

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “El segundo pilar, **Authorization**, asegura que un usuario autenticado solo ejecute las operaciones explícitamente permitidas según su rol y contexto.
> 
> **1. Control de Acceso Basado en Roles (RBAC):**
> En [`internal/auth/middleware.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/middleware.go), implementamos interceptores tanto para la API REST (`RequireRole`) como para la navegación web (`WebRequireRole`). Si una sesión con rol `vendedor` intenta invocar un endpoint administrativo, el middleware detiene la solicitud de inmediato con `403 Forbidden`.
> 
> **2. Control de Acceso Horizontal (Ownership):**
> En sistemas multi-inquilino, un error común de autorización es permitir que un usuario modifique recursos de otro (vulnerabilidad IDOR). En `MarkAsSold` ([`internal/handlers/vehicle_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/vehicle_handler.go#L76)), validamos en el backend que el `SellerID` de la publicación coincida exactamente con el `UserID` extraído de la sesión activa. Si no coinciden, se deniega con `403 Forbidden`.
> 
> **3. Máquina de Estados Estricta del Catálogo:**
> El ciclo de vida modelado en `internal/domain` garantiza que:
> * Ningún vehículo pase a venta pública sin la autorización previa del administrador.
> * Las publicaciones arrancan obligatoriamente en `PENDING`.
> * El administrador las promueve a `APPROVED`.
> * Únicamente el vendedor propietario puede marcarlas como `SOLD` tras ser aprobadas.”

---

### B. Demostración en la Interfaz de Usuario (Flujos HU-VIS-01, HU-VEN-01, HU-ADM-01)

#### [PANTALLA / ACCIÓN]
1. **Visitante (HU-VIS-01):**
   * Navegar a [`http://localhost:8080/catalog`](http://localhost:8080/catalog) sin sesión iniciada (badge *Visitante*).
   * Mostrar que el catálogo está vacío o solo muestra autos autorizados.
   * Hacer clic en un vehículo: mostrar detalle y datos de contacto del vendedor.
   * Intentar ingresar a la URL de un vehículo no aprobado: mostrar la página de error uniforme 404 (*mínima revelación*).
2. **Vendedor (HU-VEN-01):**
   * Iniciar sesión como vendedor (`/login`).
   * Redirección automática a [`/seller`](http://localhost:8080/seller).
   * Completar el formulario "Publicar Vehículo", adjuntar fotografía y enviar.
   * Mostrar en "Mis Vehículos" que la publicación aparece con badge amarillo **`PENDIENTE`**.
   * Volver a la pestaña del Catálogo Público: mostrar que el vehículo recién creado **no es visible para los visitantes**.
3. **Frontera de Autorización (Intrusión RBAC):**
   * En la barra de navegación del vendedor, intentar escribir manualmente la URL [`http://localhost:8080/admin`](http://localhost:8080/admin).
   * Mostrar la pantalla de error **`403 Acceso Denegado`** renderizada por `WebRequireRole`.
4. **Administrador (HU-ADM-01):**
   * Cerrar sesión e iniciar como `admin` -> redirige a [`/admin`](http://localhost:8080/admin).
   * En la tabla "Publicaciones Pendientes de Autorización", mostrar el vehículo enviado por el vendedor.
   * Hacer clic en el botón verde **`Aprobar`**.
   * Ir al Catálogo Público: evidenciar que ahora el vehículo **sí es visible** para todos los visitantes con su información de contacto.
5. **Cierre de Ciclo de Venta (Ownership):**
   * Volver a la sesión del vendedor (`/seller`).
   * Mostrar que ahora el vehículo está en estado **`APROBADA`** y se habilita el botón **`Marcar como Vendido`**.
   * Hacer clic en "Marcar como Vendido": el estado cambia a **`SOLD`** y desaparece del catálogo público.
6. **Autoprotección e Integridad Administrativa:**
   * En el panel de administrador, mostrar la sección "Gestión de Usuarios".
   * Mostrar que si se intenta eliminar al administrador actual, el sistema lo impide con `400 Bad Request` protegiendo la plataforma de quedar acéfala.
   * Al eliminar a un vendedor sancionado, mostrar en código `tokenStore.DeleteByUserID`: todas sus sesiones en RAM se destruyen inmediatamente en cascada.”

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “Veamos cómo la interfaz de usuario refleja con exactitud cada una de estas fronteras arquitectónicas:
> 
> * **Visitante (`HU-VIS-01`):** Un visitante consulta libremente el catálogo público. La capa de almacenamiento filtra con `FindApproved()`, garantizando que jamás se listen publicaciones en estado pendiente o vendido. Además, si el visitante intenta adivinar el ID de una publicación privada por URL, respondemos `404 Not Found` en lugar de `403`, evitando que descubra qué identificadores existen.
> * **Vendedor (`HU-VEN-01`):** Al enviar el formulario con imagen, el controlador `CreateVehicleSubmit` sanea los textos con `strings.TrimSpace`, valida que el precio y año sean coherentes, almacena la fotografía en `uploads/` con un hash único contra colisiones y path traversal, y guarda la publicación forzando el estado `PENDING`. Por tanto, el auto permanece invisible al público.
> * **Frontera de Autorización:** Si el vendedor intenta forzar el acceso a [`/admin`], nuestro middleware web intercepta la petición, verifica que `session.Role == "vendedor"` y aborta la solicitud renderizando una vista de error `403 Acceso Denegado`.
> * **Administrador (`HU-ADM-01`):** Al ingresar al panel `/admin`, la cola de moderación muestra los vehículos pendientes. Al pulsar 'Aprobar', el backend actualiza el estado a `APPROVED`, materializando la aparición del vehículo en el catálogo público junto con el contacto del vendedor.
> * **Propiedad y Retiro:** El vendedor puede marcar como vendido su auto desde su panel, actualizándolo a `SOLD` y retirándolo de la oferta pública. Si un vendedor malicioso interceptara la llamada intentando liquidar un vehículo de otro colega, el backend rechaza la acción con `403 Forbidden` gracias a la validación de propiedad horizontal.
> * **Revocación Inmediata de Sesiones:** Cuando el administrador elimina una cuenta en `/admin/users/:id/delete`, el servidor no solo borra el usuario de `UserStore`, sino que invoca `tokenStore.DeleteByUserID(id)`. Esto purga instantáneamente todos sus tokens activos en RAM, desautenticándolo en tiempo real ante cualquier petición subsecuente.”

---

## 4. Pilar 3: Accounting (Auditoría / Trazabilidad) (10:15 - 12:30)

### A. Implementación Concurrente e Inmutable en Backend

#### [PANTALLA / ACCIÓN]
* Mostrar en código [`internal/auth/audit.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/audit.go#L13-L64) destacando `AuditLogEntry`, el mutex `sync.Mutex` y las banderas de apertura `os.O_APPEND|os.O_CREATE|os.O_WRONLY`.
* Mostrar en [`cmd/server/main.go`](file:///home/nomka/Projects/AutoMarket/cmd/server/main.go#L63) el registro global del middleware: `r.Use(auth.AuditLogMiddleware(auditLogger))`.

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “El tercer pilar, **Accounting**, garantiza la trazabilidad e integridad de todas las acciones que ocurren en el ecosistema.
> 
> **Registro Global del 100% del Tráfico:**
> En [`cmd/server/main.go`](file:///home/nomka/Projects/AutoMarket/cmd/server/main.go#L63), registramos `AuditLogMiddleware` directamente sobre el motor de Gin (`r.Use`). Esto asegura que **absolutamente todas las solicitudes HTTP** —tanto públicas como privadas, exitosas o bloqueadas por seguridad— sean auditadas antes de abandonar el ciclo de vida de la petición.
> 
> **Estructura JSON Lines y Concurrencia:**
> En [`internal/auth/audit.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/audit.go):
> * Cada evento genera una estructura `AuditLogEntry` serializada en una línea JSON independiente (*JSON Lines*).
> * Registra: marca de tiempo en UTC con formato RFC3339, método HTTP, ruta solicitada, dirección IP del cliente, `user_id`, `role`, código de estado HTTP y duración de procesamiento en milisegundos (`duration_ms`).
> * Si la petición es pública, registra automáticamente `anonymous` y `visitor`. Si está autenticada, extrae la identidad del contexto inyectado por el middleware.
> * **Integridad concurrente:** La escritura en `audit.log` se sincroniza mediante un cerrojo exclusivo `sync.Mutex` y banderas de archivo `O_APPEND|O_WRONLY`, garantizando que ráfagas de peticiones concurrentes jamás corrompan las líneas ni provoquen condiciones de carrera.”

---

### B. Demostración en Tiempo Real de la Trazabilidad

#### [PANTALLA / ACCIÓN]
* Abrir una terminal en paralelo junto al navegador y ejecutar:
  ```bash
  tail -f audit.log
  ```
* En el navegador web:
  1. Recargar el catálogo público (`GET /catalog`) -> Mostrar la nueva línea con rol `visitor`, estado `200`.
  2. Intentar ingresar a `/admin` con cuenta de vendedor -> Mostrar la línea con rol `vendedor`, ruta `/admin`, estado `403`.
  3. Iniciar sesión como admin y aprobar una publicación -> Mostrar la línea `POST /admin/publications/.../approve`, rol `admin`, estado `303/200`.
* Pausar y resaltar una línea JSON completa para explicar los campos en pantalla.

#### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “Veamos la auditoría en acción en tiempo real:
> 
> Con el comando `tail -f audit.log` monitoreamos las trazas generadas por nuestras interacciones en la interfaz:
> 1. Al navegar como visitantes por el catálogo, observamos un evento con `user_id: "anonymous"`, `role: "visitor"`, método `GET /catalog` y respuesta `200 OK`.
> 2. Cuando intentamos ingresar indebidamente al panel de administrador con rol de vendedor, observamos de inmediato el registro del incidente: método `GET /admin`, el ID real del usuario, su rol `vendedor`, y el código de rechazo `403 Forbidden`.
> 3. Al aprobar el vehículo, queda registrada la auditoría administrativa con el ID del administrador, la acción `POST /admin/publications/.../approve`, y la duración exacta en milisegundos.
> 
> Este log estructurado permite reconstruir cualquier línea temporal forense, cumpliendo cabalmente el principio de no repudio y trazabilidad exigido por el estándar AAA.”

---

## 5. Verificación Automatizada E2E y Conclusión (12:30 - 14:15)

### [PANTALLA / ACCIÓN]
* En la terminal, mostrar la ejecución completa del script de pruebas de integración:
  ```bash
  ./test_flows.sh
  ```
* Dejar que el script corra en pantalla mostrando cada uno de los 9 pasos y los mensajes verdes `[PASS] HTTP ...`.
* Mostrar el paso final donde el script verifica la presencia de registros en `audit.log`.

### [LOCUCIÓN / SUSTENTACIÓN TÉCNICA]
> “Para garantizar la confiabilidad técnica de toda la plataforma, compilamos un conjunto de pruebas automatizadas de extremo a extremo en el script [`test_flows.sh`](file:///home/nomka/Projects/AutoMarket/test_flows.sh).
> 
> Como vemos en la consola, el script ejecuta secuencialmente 9 escenarios que validan toda la superficie de ataque y los requerimientos funcionales:
> * **Paso 1:** Consulta pública de catálogo (`HU-VIS-01`, código `200 OK`).
> * **Paso 2:** Registro de vendedores, bloqueo estricto ante intento de crear un segundo administrador (`403 Forbidden`) y login válido e inválido (`401 Unauthorized`).
> * **Paso 3:** Bloqueo de operaciones de vendedor sin token o con rol incorrecto (`401` y `403`).
> * **Paso 4:** Publicación de vehículos en estado `PENDING` por el vendedor (`HU-VEN-01`, código `201 Created`).
> * **Paso 5:** Aprobación de publicaciones por el administrador y paso a `APPROVED` (`HU-ADM-01`, código `200 OK`).
> * **Paso 6:** Validación de Propiedad Horizontal: rechazo con `403 Forbidden` si el vendedor 2 intenta marcar como vendido el auto del vendedor 1.
> * **Paso 7:** Cierre exitoso de venta (`SOLD`) por el vendedor legítimo y confirmación de que desaparece del catálogo público.
> * **Paso 8:** Eliminación de usuario por el administrador, bloqueo de autoeliminación (`400 Bad Request`) e invalidación inmediata de tokens en memoria (`401 Unauthorized`).
> * **Paso 9:** Verificación de integridad y persistencia del archivo `audit.log`.
> 
> **Conclusión:**
> La solución de AutoMarket demuestra que es posible implementar una arquitectura limpia, robusta, altamente concurrente y estrictamente apegada al **Marco de Seguridad AAA** utilizando únicamente las herramientas nativas de la biblioteca estándar de Go y Gin, garantizando rendimiento, mantenibilidad y ciberseguridad en cada capa.
> 
> Muchas gracias por su atención.”

---

## Resumen de Correspondencia Técnica para la Sustentación

| Pilar AAA | Requisito Funcional | Componente Backend | Implementación Técnica | Acción Demostrada en Interfaz Web |
| :--- | :--- | :--- | :--- | :--- |
| **Authentication** | RF-01 (Autenticación) | [`crypto.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/crypto.go)<br>[`token_store.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/token_store.go)<br>[`auth_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/auth_handler.go) | • Sal de 16 bytes con `crypto/rand`<br>• Hash SHA-256 (`crypto/sha256`)<br>• Tokens opacos de 32 bytes en RAM<br>• Anti-enumeración (401 unificado)<br>• Anti-Mass Assignment (RoleSeller forzado) | • Registro en `/register`<br>• Inicio de sesión en `/login`<br>• Inspección de cookie `token` con flag `HttpOnly`<br>• Bloqueo en `/setup-admin` para segundo admin |
| **Authorization** | RF-01 (Autorización)<br>HU-VIS-01<br>HU-VEN-01<br>HU-ADM-01 | [`middleware.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/middleware.go)<br>[`vehicle_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/vehicle_handler.go)<br>[`admin_handler.go`](file:///home/nomka/Projects/AutoMarket/internal/handlers/admin_handler.go) | • RBAC (`RequireRole`, `WebRequireRole`)<br>• Control Horizontal: `pub.SellerID == session.UserID`<br>• Máquina de estados: `PENDING` -> `APPROVED` -> `SOLD`<br>• Revocación en cascada (`DeleteByUserID`) | • Visitante solo ve autos `APPROVED` en `/catalog`<br>• Auto nuevo queda `PENDING` en `/seller`<br>• Vendedor recibe `403` al intentar abrir `/admin`<br>• Admin aprueba en `/admin` y aparece en catálogo<br>• Vendedor marca `SOLD` su auto |
| **Accounting** | RF-01 (Auditoría) | [`audit.go`](file:///home/nomka/Projects/AutoMarket/internal/auth/audit.go)<br>[`cmd/server/main.go`](file:///home/nomka/Projects/AutoMarket/cmd/server/main.go) | • Middleware global `AuditLogMiddleware`<br>• Formato JSON Lines en `audit.log`<br>• Registro inmutable protegido por `sync.Mutex`<br>• Trazabilidad de IP, actor, rol, método, status y ms | • Monitoreo en vivo con `tail -f audit.log`<br>• Inspección de trazas al navegar catálogo, fallar login, recibir 403 y moderar publicaciones |
