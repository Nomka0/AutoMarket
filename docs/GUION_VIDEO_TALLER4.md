# Guion de sustentación en video · Taller 4
## OWASP A07:2021 Identification and Authentication Failures / A07:2025 Authentication Failures · AutoMarket Backend

- **Duración:** 06:30
- **Grabación:** cuatro tomas continuas, una por integrante; ensamblar en orden.
- **Plataforma:** AutoMarket Backend (Go + Gin).
- **Deck:** [diapositivas_taller4.html](taller4/diapositivas_taller4.html), ocho diapositivas.

| Toma | Tiempo | Integrante | Recorrido |
|---|---:|---|---|
| 1 | 00:00–01:20 | Juan José Bolaños Delgado | Slides 1–2; navegador y estructura del backend |
| 2 | 01:20–03:00 | José Antonio Rodríguez Castelblanco | Slide 4; `crypto.go`, PoC; slide 6, benchmark |
| 3 | 03:00–04:45 | Braulio Robledo Delgado | Slide 5; remediaciones en Go |
| 4 | 04:45–06:30 | Jhon Steven Angulo Nieves | Slides 5–8; TOTP, suite, CIS, cierre y uso de IA |

## Toma 1 · Juan José · 00:00–01:20

### 00:00–00:25 · Portada · slide 1

**Pantalla:** portada. Presentar el equipo en orden de intervención: José Antonio Rodríguez Castelblanco, Braulio Robledo Delgado, Jhon Steven Angulo Nieves y Juan José Bolaños Delgado.

**Locución:**

> Cordial saludo. Bienvenidos a la sustentación técnica del Taller 4 de Ciberseguridad. Somos el equipo integrado por José Antonio Rodríguez Castelblanco, Braulio Robledo Delgado, Jhon Steven Angulo Nieves y quien les habla, Juan José Bolaños Delgado.

### 00:25–00:55 · OWASP A07 · slide 2

**Pantalla:** evolución A02:2017 → A07:2021 → A07:2025 y cuatro vectores CWE.

**Locución:**

> Abordamos fallos de identificación y autenticación. La categoría pasó de A02 en 2017 a A07 en 2021. En 2025 conserva el puesto siete y cambia su nombre a Authentication Failures. En AutoMarket evaluamos cuatro problemas: hashes rápidos, intentos de login sin límite, contraseñas débiles y falta de MFA para administradores.

### 00:55–01:20 · AutoMarket y traspaso · sin diapositiva

**Pantalla:** abrir `http://localhost:8080/catalog`; mostrar `cmd/server/`, `internal/auth/`, `internal/store/` e `internal/handlers/`.

**Locución:**

> La auditoría se realizó sobre AutoMarket, un backend en Go para compraventa de vehículos con roles de visitante, vendedor y administrador. Expondremos el antes y el después. Ahora José Antonio mostrará las fallas iniciales y su explotación.

## Toma 2 · José Antonio · 01:20–03:00

### 01:20–01:55 · Código vulnerable · slide 4

**Pantalla:** abrir `internal/auth/crypto.go` y comparar con `main`. Mostrar `HashPassword` con SHA-256 y salt. Luego abrir `internal/handlers/auth_handler.go`.

**Locución:**

> En la versión inicial encontramos dos fallas de autenticación. CWE-916: las contraseñas se guardaban con SHA-256 y salt. El salt evita tablas arcoíris, pero SHA-256 sigue siendo rápido para probar claves en paralelo. CWE-307: el login no limitaba los intentos ni bloqueaba la cuenta.

### 01:55–02:25 · Fuerza bruta en vivo · slide 4

**Pantalla:** servidor vulnerable desde `main`; ejecutar diez solicitudes fallidas:

```bash
for i in {1..10}; do
  result=$(curl -s -o /dev/null -w '%{http_code} %{time_total}' -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"wrong"}')
  http_code=${result%% *}; elapsed=${result#* }
  case "$http_code" in
    401) printf 'HTTP 401 Unauthorized en %ss\n' "$elapsed" ;;
    000) printf 'Sin respuesta HTTP (curl 000; servidor inaccesible)\n' ;;
    *) printf 'HTTP %s en %ss\n' "$http_code" "$elapsed" ;;
  esac
done
```

**Nota:** si aparece `curl 000`, el servidor no respondió; verificar que `main` esté ejecutándose en el puerto 8080.

**Locución:**

> Enviamos diez contraseñas incorrectas al backend original. Las diez solicitudes reciben HTTP 401 Unauthorized en milisegundos. No aparece el 429 ni se bloquea la cuenta. Ese comportamiento permite automatizar fuerza bruta y credential stuffing.

### 02:25–03:00 · Benchmark · slide 6

**Pantalla:** ejecutar `python3 scripts/benchmark_hash_cracking.py`; mostrar medición local de CPU y proyección GPU.

**Locución:**

> El benchmark mide cerca de medio microsegundo por hash para SHA-256 y 27 milisegundos para Argon2id en nuestro procesador. La tabla GPU es una proyección offline para claves alfanuméricas de ocho caracteres: una RTX 4090 supera 9.2 gigahashes por segundo con SHA-256, frente a unos 160 hashes por segundo con Argon2id. El tiempo medio proyectado pasa de 3.3 horas a más de 21 mil años. Braulio mostrará las defensas implementadas.

## Toma 3 · Braulio · 03:00–04:45

### 03:00–03:35 · Argon2id · slide 5

**Pantalla:** abrir `internal/auth/crypto.go`; mostrar parámetros y `subtle.ConstantTimeCompare`.

**Locución:**

> Para mitigar CWE-916 migramos a Argon2id con 64 megabytes de memoria, tres iteraciones y cuatro hilos. La comparación en tiempo constante reduce filtraciones por diferencias de temporización.

### 03:35–04:10 · Rate limit y bloqueo · slide 5

**Pantalla:** abrir `internal/auth/ratelimit.go` y `internal/store/user_store.go`; mostrar Token Bucket, `Retry-After` y `RecordFailedLogin`.

**Locución:**

> CWE-307 se mitiga en dos capas. El Token Bucket limita el login a cinco solicitudes por minuto y responde 429 al exceder la cuota. Además, después de cinco fallos, `UserStore` bloquea la cuenta durante quince minutos y responde 423.

### 04:10–04:45 · Política de contraseñas · slide 5

**Pantalla:** abrir `internal/auth/password_policy.go` y mostrar `internal/auth/seclists_top500.txt`.

**Locución:**

> La política rechaza claves de menos de diez caracteres, las que contienen el nombre de usuario y las incluidas en la lista SecLists. El registro responde 400 si la contraseña no cumple. Es una política informada por NIST SP 800-63B; no afirmamos cumplimiento integral del estándar. Ahora Steven explicará MFA y las pruebas.

## Toma 4 · Steven · 04:45–06:30

### 04:45–05:18 · MFA TOTP · slide 5

**Pantalla:** abrir `internal/handlers/totp.go` y `internal/handlers/auth_handler.go`; mostrar `generateTOTPSetup`, `verifyTOTP` y el campo `totp_code` para Admin.

**Locución:**

> Para proteger cuentas administrativas implementamos TOTP según RFC 6238. El usuario configura una semilla compatible con Google Authenticator o Aegis. El login de Admin requiere contraseña y el código vigente de seis dígitos; si falta o no es válido, se deniega el acceso con 401.

### 05:18–05:43 · Suite de seguridad · slide 6

**Pantalla:** con el servidor en `taller_4`, ejecutar `bash test_auth_security.sh` y mostrar sus cuatro resultados.

**Locución:**

> La suite cubre las cuatro defensas: 429 por exceso de solicitudes, 423 por bloqueo de cuenta, 400 ante contraseñas inseguras y 401 si Admin intenta entrar sin TOTP. Revisemos el resultado de cada prueba en la terminal.

### 05:43–06:03 · Mapeo CIS · slide 3

**Pantalla:** mostrar las salvaguardas y el alcance parcial señalado en la matriz.

**Locución:**

> CIS 5.2 trata de contraseñas únicas; nuestra política aporta controles, pero no demuestra unicidad y es parcial para cuentas sin MFA. CIS 6.3 exige MFA en aplicaciones expuestas; nuestra cobertura es parcial. CIS 6.5 es MFA administrativa. El lockout mitiga CWE-307; no es CIS 6.3.

### 06:03–06:12 · Conclusiones · slide 7

**Pantalla:** dejar visible el cierre técnico.

**Locución:**

> Un salt no vuelve lento un hash rápido. Argon2id encarece el cracking offline y las capas de acceso reducen los ataques automatizados.

### 06:12–06:30 · Autoría y uso responsable de IA · slide 8

**Pantalla:** mostrar la declaración y la captura del encargo. Señalar ClickUp, revisión por pares y responsabilidad del equipo.

**Locución:**

> ClickUp organizó las doce tareas de este Taller 4 por responsable y prioridad. El plan de trabajo establece revisión por pares mediante Pull Request antes de integrar cambios. Usamos IA para apoyar la síntesis y el diseño; la responsabilidad técnica y académica sigue siendo del equipo. Muchas gracias.

## Ficha rápida por integrante

1. **Juan José:** slide 1 → slide 2 → `/catalog` y estructura Go.
2. **José Antonio:** slide 4 → comparación `crypto.go` con `main` → PoC → slide 6 y benchmark.
3. **Braulio:** slide 5 → `crypto.go` → `ratelimit.go` → `user_store.go` → `password_policy.go`.
4. **Steven:** slide 5 → TOTP → slide 6 y pruebas → slide 3, CIS → slide 7, conclusiones → slide 8, declaración IA.
