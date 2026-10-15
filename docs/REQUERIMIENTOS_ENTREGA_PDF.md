# AutoMarket - Requerimientos y Estructura del Documento PDF Final

Guía de especificaciones y lista de chequeo para la consolidación del **único archivo PDF de entrega** del taller de Ciberseguridad (US-16).

---

## 1. Reglas Estrictas del Enunciado

* **Formato**: Un único archivo `.pdf`.
* **Docente / Revisor técnico**: Iván M. Cabezas T. (`imcabezas`).
* **Límite de alcance**: Ceñirse estrictamente a lo pedido en las HUs y en el RF-01. No inventar endpoints ni capacidades no solicitadas.

---

## 2. Secciones Obligatorias del PDF

### 1. Portada Institucional
* **Título del proyecto**: Taller: Implementación de AAA en sitio web - AutoMarket Backend.
* **Asignatura**: Ciberseguridad.
* **Docente**: Iván M. Cabezas T.
* **Integrantes (Obligatorio en orden alfabético por Apellido - Nombre)**:
  1. Angulo Nieves, Jhon Steven
  2. Bolaños Delgado, Juan José
  3. Giraldo, Javier
  4. Robledo Delgado, Braulio
* **Fecha de entrega**: Octubre 2026.

---

### 2. Información del Repositorio y Versión
* **URL del repositorio GitHub**: `https://github.com/Nomka0/AutoMarket`
* **Hash del commit de entrega (HEAD de `main`)**:
  * Ejecutar en terminal: `git rev-parse HEAD`
* **Confirmación de invitación a colaborador**:
  * Captura de pantalla o confirmación de la invitación enviada al usuario GitHub: **`imcabezas`** (US-17).

---

### 3. Sustentación y Video
* **URL pública del video de demostración**: Enlace a YouTube (no listado/público) o Google Drive (con permisos abiertos de lectura).
* **Contenido mínimo que debe evidenciar el video**:
  * Explicación de la arquitectura y confirmación de dependencias en `go.mod` (solo Gin).
  * Demostración en vivo de los flujos AAA y las 3 Historias de Usuario ejecutando `./test_flows.sh`.
  * Verificación del archivo `audit.log` generado cronológicamente.
  * Participación de los integrantes del equipo.

---

### 4. Diagramas de Arquitectura (US-14)

El documento debe incluir exactamente **dos (2) diagramas**:

#### A. Diagrama de Estructura (Componentes / Paquetes Go)
* **Objetivo**: Mostrar la separación de responsabilidades y bajo acoplamiento.
* **Capas a ilustrar**:
  * `cmd/server/main.go` (Punto de entrada y cableado de dependencias).
  * `internal/handlers` (Controladores Gin: `AuthHandler`, `VehicleHandler`, `AdminHandler`, `CatalogHandler`).
  * `internal/auth` (Seguridad: `crypto.go`, `TokenStore`, `AuthMiddleware`, `RequireRole`, `AuditLogMiddleware`).
  * `internal/store` (Almacenes thread-safe con persistencia JSON stdlib: `UserStore`, `VehicleStore`, `TokenStore`).
  * `internal/domain` (Entidades puras: `User`, `VehiclePublication`).
  * Salidas a archivo: `audit.log` (Accounting) y `data/*.json` (Persistencia duradera).

#### B. Diagrama de Comportamiento (Diagrama de Secuencia AAA)
* **Objetivo**: Ilustrar la traza de una petición protegida a través del pipeline AAA.
* **Actores y componentes**:
  * Cliente (Vendedor) ➔ Gin Engine ➔ `AuditLogMiddleware` ➔ `AuthMiddleware` ➔ `RequireRole` ➔ `VehicleHandler` ➔ `VehicleStore`.
* **Eventos clave**:
  1. Registro de inicio de tiempo (`start := time.Now()`).
  2. Validación de token Bearer en `TokenStore` (Authentication - 401 si falla).
  3. Comprobación de rol de vendedor (Authorization - 403 si falla).
  4. Ejecución del handler y persistencia del vehículo en estado `PENDING`.
  5. Retorno de respuesta HTTP 201 Created.
  6. Cálculo de latencia (`duration_ms`) y escritura síncrona en `audit.log` (Accounting).

---

### 5. Resumen Técnico del Marco AAA y Cumplimiento de HUs
* **Authentication**: Explicar la sustitución de JWT/bcrypt por funciones nativas de la biblioteca estándar (`crypto/rand` y `crypto/sha256`) con Salt único por usuario y tokens aleatorios de 32 bytes con expiración de 24 horas.
* **Authorization**: Explicar el middleware RBAC (`RequireRole`) y el control horizontal de recursos (un vendedor solo puede marcar como vendido su propio vehículo).
* **Accounting**: Describir el `AuditLogMiddleware` y la estructura de los registros JSON Lines en `audit.log`.
* **Persistencia y Protección de Datos en Disco**: Explicar la persistencia sin librerías externas mediante archivos JSON locales (`data/users.json`, `data/vehicles.json`), la coordinación de concurrencia con `sync.RWMutex` para evitar condiciones de carrera, y la separación estricta entre entidades de dominio (que persisten hash/salt) y DTOs de salida HTTP (que impiden fugas de credenciales).
* **Historias de Usuario**:
  * `HU-VIS-01`: Catálogo público accesible a visitantes sin token; aislamiento de vehículos pendientes y vendidos con respuesta uniforme `404 Not Found`.
  * `HU-VEN-01`: Publicación forzada en `PENDING` y reporte de venta (`SOLD`).
  * `HU-ADM-01`: Moderación de publicaciones (`PENDING` ➔ `APPROVED`), eliminación de publicaciones y eliminación de usuarios con revocación de tokens.

---

### 6. Declaración de Uso Responsable de IA
* **Texto formal obligatorio**:
  * Declarar explícitamente el rol de las herramientas de inteligencia artificial utilizadas durante el proyecto (ej. asistencia en diseño de pruebas, revisión de código, optimización de documentación).
  * Afirmación explícita de que el equipo comprende, verificó y es responsable del 100% del código fuente entregado.

---

## 3. Checklist Previo a la Generación del PDF

- [x] Orden alfabético estricto de integrantes por Apellido: Angulo Nieves, Bolaños Delgado, Giraldo, Robledo Delgado.
- [x] Commit hash verificado en la rama `main` de GitHub.
- [ ] Usuario `imcabezas` invitado formalmente como colaborador al repositorio.
- [ ] Video grabado, subido y con enlace verificado en modo incógnito (sin restricciones de acceso).
- [x] Diagrama 1: Estructura de paquetes Go incluido con alta resolución.
- [x] Diagrama 2: Secuencia del pipeline AAA incluido con alta resolución.
- [x] Declaración de uso responsable de IA redactada y firmada/declarada por el equipo.
- [ ] Archivo exportado y nombrado formalmente en formato `.pdf`.
