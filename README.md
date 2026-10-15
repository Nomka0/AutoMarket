# AutoMarket 🚗

Plataforma web en Go para la compra y venta segura de vehículos usados. El proyecto implementa el marco de seguridad **AAA** (*Authentication, Authorization, Accounting*) y las 3 Historias de Usuario del taller sin usar librerías externas adicionales.

---

## 👥 Integrantes (Equipo)
* **Angulo Nieves, Jhon Steven**
* **Bolaños Delgado, Juan José**
* **Giraldo, Javier**
* **Robledo Delgado, Braulio**

* **Docente:** Iván M. Cabezas T.  
* **Materia:** Ciberseguridad  

---

## 🚀 Cómo arrancar el proyecto

1. **Clonar el repo y entrar:**
   ```bash
   git clone https://github.com/Nomka0/AutoMarket.git
   cd AutoMarket
   ```

2. **Prender el servidor:**
   ```bash
   go run ./cmd/server
   ```
   El servidor queda listo escuchando en `http://localhost:8080`.
   *(Opcional: usar `STORAGE_MODE=memory go run ./cmd/server` para arrancar en memoria pura sin tocar disco).*

3. **Correr las pruebas:**
   * Pruebas unitarias de Go:
     ```bash
     go test ./...
     ```
   * Prueba de todos los flujos de punta a punta (con el server corriendo en otra terminal):
     ```bash
     ./test_flows.sh
     ```

---

## 🖥️ Cómo usar la plataforma (Roles)

Abre en tu navegador `http://localhost:8080/catalog`:

* **Visitante (Público):** Entras directo sin cuenta. Puedes mirar los autos publicados y ver el contacto del vendedor. No ves autos pendientes ni vendidos.
* **Vendedor:**
  1. Haces clic en **Registrarse** (`/register`), pones tu usuario, clave y datos de contacto.
  2. Inicias sesión en `/login`.
  3. Desde **Mi Panel** publicas un auto. La publicación queda en estado **Pendiente** hasta que el admin la revise.
  4. Cuando vendas el auto, lo marcas como vendido y se quita automáticamente del catálogo.
* **Administrador:**
  1. Si es la primera vez que corre la app y no hay admin, te saldrá el botón discreto **Setup Admin** arriba a la derecha (o vas directo a `/setup-admin`). Solo deja crear un administrador.
  2. También puedes precargarlo creando un archivo `.env` con `ADMIN_USERNAME` y `ADMIN_PASSWORD`.
  3. Desde el **Panel Admin** (`/admin`), autorizas o rechazas autos pendientes y puedes eliminar cuentas si es necesario.

---

## 📁 Archivos clave y qué hace cada uno

* `data/`: Carpeta local donde se guardan `users.json` y `vehicles.json`. Permite que no se borren los autos ni usuarios al reiniciar el server. Está en el `.gitignore` para no subir datos de prueba al repositorio.
* `audit.log`: Archivo donde se anota cada petición que entra al backend (quién fue, qué rol tenía, qué ruta tocó, código HTTP y cuánto tardó en milisegundos). Esto cumple el requisito de *Accounting* (auditoría y trazabilidad).
* `templates/` y `static/`: Contienen las vistas HTML y estilos CSS con diseño oscuro (*Showroom Titanium & Crimson Racing*).
* `cmd/server/main.go`: Punto de entrada que arranca Gin, lee las plantillas y conecta todo.
* `internal/`:
  * `auth/`: La seguridad (generación de Salt, hashing de contraseñas, tokens de sesión y middlewares).
  * `store/`: Almacenes concurrentes (`sync.RWMutex`) con guardado a JSON.
  * `handlers/`: Los controladores que procesan las peticiones web y de la API.
  * `domain/`: Los modelos de datos (`User`, `VehiclePublication`).

---

## 🔒 Cosas importantes que pide el taller (Ciberseguridad)

* **Solo Gin en `go.mod`:** Como pidió el profe, `github.com/gin-gonic/gin` es la **única** dependencia externa directa permitida.
* **Criptografía sin librerías raras:** Nada de usar JWT ni bcrypt. Todo el manejo de claves se hizo con la librería estándar de Go:
  * Hashes con `crypto/sha256`.
  * **Salt único:** Cada usuario tiene 16 bytes aleatorios generados con `crypto/rand` para evitar ataques de tablas arcoíris.
  * Comparación de contraseñas en tiempo constante (`subtle.ConstantTimeCompare`) para evitar ataques de temporización (*timing attacks*).
* **Control de acceso (RBAC):** Nadie entra a rutas de vendedor o admin sin token válido, y un vendedor solo puede marcar como vendido su propio vehículo.
* **Persistencia segura y sin fugas de datos:** La persistencia a `data/users.json` y `data/vehicles.json` se logra únicamente con la biblioteca estándar (`encoding/json`, `os`). Aunque el archivo en disco guarda `password_hash` y `salt`, la API pública jamás los expone porque los controladores devuelven DTOs explícitos.
* **Sesiones volátiles en memoria:** Los tokens de sesión viven exclusivamente en RAM con TTL (24h); al apagar el servidor se invalidan limpiamente y nunca quedan tokens activos expuestos en archivos del disco.
