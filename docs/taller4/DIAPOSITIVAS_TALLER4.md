# Diapositivas de Sustentación Técnica: Taller 4
## OWASP Top 10 - A07: Identification and Authentication Failures en AutoMarket

---


## Orden de grabación

Graben cuatro tomas continuas por separado y únanlas en edición, en este orden:

| Toma | Integrante | Diapositivas | Tiempo |
|---|---|---|---:|
| 1 | Juan José Bolaños | 1-3 | 00:00-01:20 |
| 2 | José Antonio Rodríguez Castelblanco | 4-5 | 01:20-03:00 |
| 3 | Braulio Robledo | 6 | 03:00-04:45 |
| 4 | Jhon Steven Angulo | 7-9 | 04:45-06:30 |

Cada integrante prepara el servidor y los archivos antes de grabar. Mantengan una sola toma por persona; las pausas para cambiar de rama o iniciar el backend ocurren antes de grabar.

---

<!-- SLIDE 1 -->
# [DIAPOSITIVA 1] Portada Institucional y Presentación del Proyecto

### **UNIVERSIDAD DEL VALLE**
**Facultad de Ingeniería - Escuela de Ingeniería de Sistemas y Computación**
**Asignatura:** Ciberseguridad  
**Docente:** Iván M. Cabezas T.  

---

## **Auditoría AppSec y Remediación Defensiva: OWASP A07 en AutoMarket Backend**
*Implementación de Criptografía Robusta (Argon2id), Control de Tasa (Token Bucket), Bloqueo de Cuentas y MFA TOTP (RFC 6238) bajo CIS Controls v8.1.2*

---

### **Integrantes del Equipo Evaluador (Orden Alfabético por Apellidos):**
1. **Angulo Nieves, Jhon Steven** - `jhon.angulo@correounivalle.edu.co`
2. **Bolaños Delgado, Juan José** - `juan.jose.bolanos@correounivalle.edu.co`
3. **Robledo Delgado, Braulio** - `braulio.robledo@correounivalle.edu.co`
4. **Rodríguez Castelblanco, José Antonio** - `jose.castelblanco@correounivalle.edu.co`

* **Repositorio Oficial:** `https://github.com/Nomka0/AutoMarket` (Rama: `taller_4`)
* **Fecha de Sustentación:** Noviembre 2026

---

<!-- SLIDE 2 -->
# [DIAPOSITIVA 2] Marco Teórico: ¿Qué es OWASP A07 y sus Vectores Críticos?

### 1. Evolución Histórica: de A02:2017 a A07:2021/2025
* **OWASP 2017 (`A02 - Broken Authentication`):** Foco híbrido en fallos de sesión y gestión básica de cookies/tokens.
* **OWASP 2021/2025 (`A07 - Identification and Authentication Failures`):**
  * Desciende de la posición #2 a la #7 por la adopción de proveedores centralizados de identidad (IdPs, OAuth 2.0/OIDC).
  * Separa estrictamente **Identificación** (*quién dice ser el sujeto*) de **Autenticación** (*prueba criptográfica de la identidad*).

### 2. Los 4 Vectores Críticos Identificados y Evaluados (CWEs)

| CWE Asociado | Nombre Técnico | Vector de Explotación | Impacto en AutoMarket |
| :--- | :--- | :--- | :--- |
| **CWE-916** / CWE-328 | *Insufficient Computational Effort* | Hashing no adaptativo (SHA-256 plano sin factor de coste). | Cracking masivo offline por GPU ante volcado de base de datos. |
| **CWE-307** | *Improper Restriction of Auth Attempts* | Ausencia de Rate Limiting y Lockout en `/api/auth/login`. | Credential Stuffing y fuerza bruta automatizada ilimitada. |
| **CWE-521** | *Weak Password Requirements* | Admisión de contraseñas cortas o triviales en el registro. | Compromiso inmediato de cuentas de vendedores y usuarios. |
| **CWE-308** / CWE-287 | *Improper Authentication (No MFA)* | Acceso administrativo restringido a un único factor débil (contraseña). | *Account Takeover* (ATO) total de la plataforma por toma del rol Admin. |

---

<!-- SLIDE 3 -->
# [DIAPOSITIVA 3] AutoMarket: sistema evaluado

- Backend de compraventa de vehículos construido con Go y Gin.
- Rutas de referencia: `/catalog` y `/api/auth/login`.
- Entrada: `cmd/server/`; lógica distribuida entre `internal/handlers/`, `internal/auth/` e `internal/store/`.
- Roles: visitante, vendedor y administrador.
- Para la demostración del catálogo, iniciar la rama `taller_4` antes de grabar:
  ```bash
  git switch taller_4
  go run ./cmd/server
  ```

---

<!-- SLIDE 4 -->
# [DIAPOSITIVA 4] Diagnóstico Inicial de AutoMarket: Escenario Vulnerable ("Antes")

### 1. Hallazgos en el Código Fuente Base (`main` / `036d03b`)
* **`internal/auth/crypto.go`:** Función hash `sha256.Sum256([]byte(password + salt))`. Insegura frente a hardware masivo; comparación `==` propensa a ataques de temporización (*timing attacks*).
* **`internal/handlers/auth_handler.go`:** Endpoint `/api/auth/login` validando en bucle sin latencia artificial, sin rate limiting y sin bloqueo.

### 2. Evidencia de Explotación Práctica (PoC)

* **Preparación:** en un terminal, desde la rama `main`, iniciar el servidor vulnerable y dejarlo activo; ejecutar el bucle desde otro terminal:
  ```bash
  git switch main
  go run ./cmd/server
  ```
* Si `curl` muestra `000`, no recibió una respuesta HTTP: el servidor no está disponible en `localhost:8080`.

```bash
# Ráfaga ofensiva de fuerza bruta (10 intentos en 0.02 segundos sin restricción)
for i in {1..10}; do
  result=$(curl -s -o /dev/null -w '%{http_code} %{time_total}' -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"wrong"}')
  http_code=${result%% *}; elapsed=${result#* }
  case "$http_code" in
    401) printf 'HTTP 401 Unauthorized en %ss\n' "$elapsed" ;;
    000) printf 'Sin respuesta HTTP (curl 000; servidor inaccesible)\n' ;;
    *) printf 'HTTP %s en %ss\n' "$http_code" "$elapsed" ;;
  esac
done
# Esperado: diez HTTP 401 Unauthorized; todas respondidas sin rate limit ni lockout.
```

```
+---------------------------------------------------------------------------------------+
|  [ESPACIO PARA CAPTURA DE PANTALLA 1]                                                 |
|  Terminal ejecutando el ataque de fuerza bruta con curl recibiendo 401 sin bloqueo    |
+---------------------------------------------------------------------------------------+
|  [ESPACIO PARA CAPTURA DE PANTALLA 2]                                                 |
|  Editor VS Code mostrando crypto.go vulnerable (SHA-256 plano sin factor de coste)    |
+---------------------------------------------------------------------------------------+
```

---

<!-- SLIDE 5 -->
# [DIAPOSITIVA 5] Benchmark de cracking offline

### 1. Benchmark Forense de Ruptura en Hardware Ofensivo (GPU Hashcat)
> *Espacio de claves para 8 caracteres alfanuméricos (`62^8` = 2.18 × 10¹⁴ combinaciones)*

| Hardware Ofensivo | Algoritmo Evaluado | Tasa de Procesamiento | Tiempo Medio de Ruptura (50%) | Factor de Mitigación |
| :--- | :--- | :--- | :--- | :--- |
| **NVIDIA RTX 3080** | **SHA-256 plano** | $5.2\text{ GH/s}$ | **5.83 horas** | *Crítico (Vulnerable)* |
| *(10 GB VRAM)* | **Argon2id OWASP** | $0.075\text{ kH/s}$ | **46,156 años** | **$69,300,000\times$ más lento** |
| **NVIDIA RTX 4090** | **SHA-256 plano** | $9.2\text{ GH/s}$ | **3.30 horas** | *Crítico (Vulnerable)* |
| *(24 GB VRAM)* | **Argon2id OWASP** | $0.160\text{ kH/s}$ | **21,636 años** | **$57,500,000\times$ más lento** |
| **Cluster 8x RTX 4090** | **SHA-256 plano** | $73.6\text{ GH/s}$ | **24.7 minutos** | *Extremo (Compromiso total)* |
| *(Granja de ataque)* | **Argon2id OWASP** | $1.280\text{ kH/s}$ | **2,704 años** | **Matemáticamente inviable** |

---

<!-- SLIDE 6 -->
# [DIAPOSITIVA 6] Arquitectura de Remediación Defensiva ("Después")

### Esquema de Protección en Profundidad de AutoMarket

```
 Petición Entrante (POST /api/auth/login)
            │
            ▼
 ┌──────────────────────────────────────────────┐
 │ 1. RateLimitMiddleware (Token Bucket por IP) │ ──> [Excede 5 req/min] ──> HTTP 429 Too Many Requests
 └──────────────────────┬───────────────────────┘
                        │ (Bajo la cuota)
                        ▼
 ┌──────────────────────────────────────────────┐
 │ 2. Verificación Account Lockout (UserStore)  │ ──> [Bloqueado / >5 fallos] ──> HTTP 423 Locked (15 min)
 └──────────────────────┬───────────────────────┘
                        │ (Cuenta habilitada)
                        ▼
 ┌──────────────────────────────────────────────┐
 │ 3. Verificación Hash Criptográfico           │ ──> [Falla credencial] ──> RecordFailedLogin()
 │    Argon2id (m=64MB, t=3, p=4) + ConstantTime│                            HTTP 401 Unauthorized
 └──────────────────────┬───────────────────────┘
                        │ (Contraseña correcta)
                        ▼
 ┌──────────────────────────────────────────────┐
 │ 4. Verificación de Segundo Factor (MFA)      │ ──> [Es Admin y falta/erróneo TOTP]
 │    TOTP RFC 6238 (Google Authenticator)      │     HTTP 401 Unauthorized
 └──────────────────────┬───────────────────────┘
                        │ (TOTP válido o rol no-admin)
                        ▼
           HTTP 200 OK + Sesión Bearer Token
```

### Componentes Técnicos Remediados

1. **`internal/auth/crypto.go`:** Migración a `golang.org/x/crypto/argon2` ($m=64\text{ MB}$, $t=3$, $p=4$) con `subtle.ConstantTimeCompare`.
2. **`internal/auth/ratelimit.go`:** Middleware Token Bucket en memoria (5 req/minuto por IP con cabecera `Retry-After`).
3. **`internal/store/user_store.go`:** Lógica de bloqueo temporal (5 intentos fallidos $\rightarrow$ 15 minutos en estado Locked).
4. **`internal/auth/password_policy.go`:** Validador NIST SP 800-63B en endpoints de registro y setup (`HTTP 400`).
5. **`internal/handlers/totp.go`:** Inscripción y validación estricta de 6 dígitos RFC 6238 para rol Administrador.

---

<!-- SLIDE 7 -->
# [DIAPOSITIVA 7] Pruebas automatizadas de seguridad

### Suite automatizada (`test_auth_security.sh`)

* **Preparación:** con el servidor de `main` detenido, desde la rama `taller_4` iniciar el backend y dejarlo activo en un terminal; correr la suite en otro:
  ```bash
  git switch taller_4
  go run ./cmd/server
  ```

```
======================================================================
  RESUMEN DE PRUEBAS DE SEGURIDAD (TALLER 4 - OWASP A07)
======================================================================
  ✔ [PASS] Prueba 1: Rate Limiting (HTTP 429 Too Many Requests)
  ✔ [PASS] Prueba 2: Account Lockout tras 5 intentos fallidos (HTTP 423 Locked)
  ✔ [PASS] Prueba 3: Validación de Contraseña Débil NIST (HTTP 400 Bad Request)
  ✔ [PASS] Prueba 4: Exigencia de MFA TOTP para rol Admin (HTTP 401 sin código)
----------------------------------------------------------------------
  Resultado: 4 de 4 pruebas aprobadas (100% de efectividad defensiva)
======================================================================
```

---

<!-- SLIDE 8 -->
# [DIAPOSITIVA 8] Conclusiones y mapeo CIS

### Lecciones técnicas
1. Un salt evita tablas arcoíris, pero no ralentiza SHA-256 frente al cracking paralelo.
2. Argon2id aumenta el coste de cada intento offline.
3. Rate limit, lockout y MFA cubren controles distintos del ciclo de acceso.

### Alineación con CIS Controls v8.1.2

| Referencia | Alcance en AutoMarket |
|---|---|
| CIS 5.2 · contraseñas únicas | La política local filtra claves débiles; no demuestra unicidad y la alineación es parcial. |
| CWE-307 · intentos de autenticación | Token Bucket y lockout son mitigaciones locales. El lockout no se atribuye a CIS 6.3. |
| CIS 6.3 · MFA en aplicaciones expuestas | El TOTP implementado solo para Admin ofrece cobertura parcial si la aplicación está expuesta. |
| CIS 6.5 · MFA administrativa | TOTP RFC 6238 para el rol Admin. |

**Cierre:** seguridad por diseño exige controles verificables en cada etapa del acceso.

---

<!-- SLIDE 9 -->
# [DIAPOSITIVA 9] Autoría y uso responsable de IA

- Se usó asistencia de IA para sintetizar el brief y el guion, proponer la estructura narrativa, preparar diagramas y maquetar las diapositivas. La captura de la diapositiva conserva el encargo de diseño y las fuentes indicadas.
- ClickUp: 12 tareas de Taller 4 organizadas por responsable, prioridad y tres sprints en el plan de trabajo.
- La guía de trabajo documenta revisión manual por pares mediante Pull Request antes de integrar cambios a `main`.
- El equipo revisa resultados y afirmaciones, y asume la responsabilidad técnica y académica de la entrega.
- Las tasas GPU son proyecciones offline; la medición CPU local se presenta por separado.

**Repositorio:** [Nomka0/AutoMarket](https://github.com/Nomka0/AutoMarket/tree/taller_4) · **Pruebas:** [`test_auth_security.sh`](https://github.com/Nomka0/AutoMarket/blob/taller_4/test_auth_security.sh)

---
