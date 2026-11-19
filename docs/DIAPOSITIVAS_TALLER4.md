# Diapositivas de Sustentación Técnica: Taller 4
## OWASP Top 10 — A07: Identification and Authentication Failures en AutoMarket

---

<!-- SLIDE 1 -->
# [DIAPOSITIVA 1] Portada Institucional y Presentación del Proyecto

### **UNIVERSIDAD DEL VALLE**
**Facultad de Ingeniería — Escuela de Ingeniería de Sistemas y Computación**  
**Asignatura:** Ciberseguridad  
**Docente:** Iván M. Cabezas T.  

---

## **Auditoría AppSec y Remediación Defensiva: OWASP A07 en AutoMarket Backend**
*Implementación de Criptografía Robusta (Argon2id), Control de Tasa (Token Bucket), Bloqueo de Cuentas y MFA TOTP (RFC 6238) bajo CIS Controls v8.1.2*

---

### **Integrantes del Equipo Evaluador (Orden Alfabético por Apellidos):**
1. **Angulo Nieves, Jhon Steven** — `jhon.angulo@correounivalle.edu.co`
2. **Bolaños Delgado, Juan José** — `juan.jose.bolanos@correounivalle.edu.co`
3. **Robledo Delgado, Braulio** — `braulio.robledo@correounivalle.edu.co`
4. **Rodríguez Castelblanco, José Antonio** — `jose.castelblanco@correounivalle.edu.co`

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
# [DIAPOSITIVA 3] Mapeo Normativo: Alineación con CIS Controls v8.1.2

> *Las remediaciones no constituyen parches ad-hoc; responden directamente a los controles internacionales de ciberseguridad defensiva.*

```
                 MARCO DE SEGURIDAD DEFENSIVA (CIS CONTROLS v8.1.2)
       ┌─────────────────────────────────────┬─────────────────────────────────────┐
       │     CIS Control 5: Gestión de       │    CIS Control 6: Gestión del       │
       │       Cuentas (Account Mgmt)        │   Control de Acceso (Access Ctrl)   │
       └──────────────────┬──────────────────┴──────────────────┬──────────────────┘
                          │                                     │
                 Salvaguarda 5.2                       Salvaguarda 6.3 & 6.5
                          ▼                                     ▼
             [ NIST SP 800-63B Compliant ]         [ Defensas Activas de Acceso ]
             • Longitud mínima: 10 chars           • 6.3: Account Lockout (5 intentos)
             • Filtro SecLists Top 500             • Rate Limiting (5 req/min IP)
             • Sin datos contextuales (username)   • 6.5: MFA Obligatorio Admin (TOTP)
```

### Matriz de Cumplimiento Técnico

| CIS Control | Salvaguarda | Requisito Formal CIS | Implementación Técnica en AutoMarket |
| :--- | :--- | :--- | :--- |
| **Control 5** *(Account Management)* | **5.2** | *Use Unique Passwords:* Exigir longitud y verificar contra listas de contraseñas vulneradas. | Validador en `password_policy.go` (min 10 chars, blacklist SecLists, rechazo `HTTP 400`). |
| **Control 6** *(Access Control)* | **6.3** | *Lockout Failed Attempts:* Bloquear temporalmente cuentas tras reiterados intentos fallidos. | Contador en `user_store.go` (5 fallos consecutivos $\rightarrow$ bloqueo 15 min con `HTTP 423`). |
| **Control 6** *(Access Control)* | **6.5** | *Require MFA for Administrative Access:* Exigir segundo factor en todas las cuentas privilegiadas. | TOTP RFC 6238 en `totp.go` y login condicional obligatorio para rol Admin con `HTTP 401`. |

---

<!-- SLIDE 4 -->
# [DIAPOSITIVA 4] Diagnóstico Inicial de AutoMarket: Escenario Vulnerable ("Antes")

### 1. Hallazgos en el Código Fuente Base (`main` / `036d03b`)
* **`internal/auth/crypto.go`:** Función hash `sha256.Sum256([]byte(password + salt))`. Insegura frente a hardware masivo; comparación `==` propensa a ataques de temporización (*timing attacks*).
* **`internal/handlers/auth_handler.go`:** Endpoint `/api/auth/login` validando en bucle sin latencia artificial, sin rate limiting y sin bloqueo.

### 2. Evidencia de Explotación Práctica (PoC)

```bash
# Ráfaga ofensiva de fuerza bruta (10 intentos en 0.02 segundos sin restricción)
$ for i in {1..10}; do curl -s -w "%{http_code} en %{time_total}s\n" -X POST \
    http://localhost:8080/api/auth/login -d '{"username":"admin","password":"wrong"}'; done
401 en 0.0021s
401 en 0.0019s
401 en 0.0018s
... (Todas aceptadas y respondidas inmediatamente; el servidor nunca bloquea ni limita)
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
# [DIAPOSITIVA 5] Arquitectura de Remediación Defensiva ("Después")

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

<!-- SLIDE 6 -->
# [DIAPOSITIVA 6] Resultados Cuantitativos: Benchmarks y Pruebas

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

### 2. Resultados de la Suite Automatizada (`test_auth_security.sh`)

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

<!-- SLIDE 7 -->
# [DIAPOSITIVA 7] Conclusiones, Lecciones Aprendidas y Enlaces

### Conclusiones Técnicas
1. **El salteado no salva a un mal algoritmo:** Añadir un *salt* criptográfico a SHA-256 neutraliza tablas arcoíris (*rainbow tables*), pero no ofrece ninguna resistencia contra ataques de fuerza bruta paralela en GPUs.
2. **La dureza en memoria (*Memory-Hardness*) es indispensable:** Al imponer 64 MB por operación, Argon2id satura el ancho de banda y la memoria VRAM de las GPUs, neutralizando la ventaja computacional del atacante.
3. **Defensa en profundidad en el ciclo de vida del acceso:** La combinación de Rate Limiting (red/IP) + Account Lockout (usuario/estado) + MFA (identidad) mitiga completamente los vectores automatizados de Credential Stuffing.

### Declaración de Ética y Uso Responsable de IA
* Se emplearon asistentes de IA agéntica (Antigravity CLI / Claude 3.5 Sonnet / Gemini 3.8) bajo supervisión humana para la generación de benchmarks teóricos, modelado matemático de claves y documentación técnica bajo estándares OWASP y CIS Controls.

---

### Enlaces Oficiales del Proyecto
* **Código Fuente y Pull Requests:** [GitHub: Nomka0/AutoMarket (Branch `taller_4`)](https://github.com/Nomka0/AutoMarket/tree/taller_4)
* **Suite de Pruebas Automatizadas:** [`test_auth_security.sh`](https://github.com/Nomka0/AutoMarket/blob/taller_4/test_auth_security.sh)
* **Video de Sustentación en YouTube:** `[ENLACE_AL_VIDEO_AQUÍ]`
