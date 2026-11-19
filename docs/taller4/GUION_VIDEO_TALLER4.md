# Guion de sustentación en video · Taller 4
## OWASP A07 · AutoMarket Backend

- **Duración total:** 06:30
- **Grabación:** cuatro clips independientes, uno por integrante; cada persona graba su bloque completo en una sola toma. Unir los clips en edición.
- **Deck:** [diapositivas_taller4.html](taller4/diapositivas_taller4.html), nueve diapositivas.
- **Servidor:** iniciar o cambiar de rama antes de grabar cada toma. Si se cambia de rama, detener primero el servidor con `Ctrl+C`.

| Toma | Integrante                          | Diapositivas |      Tiempo |
| ---- | ----------------------------------- | -----------: | ----------: |
| 1    | Juan José Bolaños Delgado           |          1-3 | 00:00-01:20 |
| 2    | José Antonio Rodríguez Castelblanco |          4-5 | 01:20-03:00 |
| 3    | Braulio Robledo Delgado             |            6 | 03:00-04:45 |
| 4    | Jhon Steven Angulo Nieves           |          7-9 | 04:45-06:30 |

## Toma 1 · Juan José · 00:00-01:20 · diapositivas 1-3

**Antes de grabar:** desde la rama `taller_4`, iniciar el servidor en una terminal y dejarlo activo:

```bash
git switch taller_4
go run ./cmd/server
```

Abrir `http://localhost:8080/catalog` y tener visible el árbol de paquetes para la diapositiva 3.

### 00:00-00:25 · Portada · diapositiva 1

**Locución:**

> Cordial saludo. Bienvenidos a la sustentación técnica del Taller 4 de Ciberseguridad. Somos el equipo integrado por José Antonio Rodríguez Castelblanco, Braulio Robledo Delgado, Jhon Steven Angulo Nieves y quien les habla, Juan José Bolaños Delgado.

### 00:25-00:55 · OWASP A07 · diapositiva 2

**Pantalla:** evolución A02:2017 → A07:2021 → A07:2025 y los cuatro vectores CWE.

**Locución:**

> Abordamos fallos de identificación y autenticación. La categoría pasó de A02 en 2017 a A07 en 2021. En 2025 conserva el puesto siete y cambia su nombre a Authentication Failures. En AutoMarket evaluamos cuatro problemas: hashes rápidos, intentos de login sin límite, contraseñas débiles y falta de MFA para administradores.

### 00:55-01:20 · AutoMarket y traspaso · diapositiva 3

**Pantalla:** mostrar `/catalog`, `cmd/server/`, `internal/handlers/`, `internal/auth/` e `internal/store/`.

**Locución:**

> La auditoría se realizó sobre AutoMarket, un backend en Go para compraventa de vehículos con roles de visitante, vendedor y administrador. Expondremos el antes y el después. Ahora José Antonio mostrará las fallas iniciales y su explotación.

## Toma 2 · José Antonio Rodríguez Castelblanco · 01:20-03:00 · diapositivas 4-5

**Antes de grabar:** detener cualquier servidor anterior. Desde `main`, iniciar el backend vulnerable en una terminal y dejarlo activo durante la toma:

```bash
git switch main
go run ./cmd/server
```

Tener un segundo terminal listo para ejecutar la PoC y el benchmark. No hace falta detener la grabación entre las diapositivas 4 y 5.

### 01:20-01:55 · Código vulnerable · diapositiva 4

**Pantalla:** comparar `internal/auth/crypto.go` de `main` con la versión de `taller_4`. Mostrar SHA-256 con salt y abrir `internal/handlers/auth_handler.go`.

**Locución:**

> En la versión inicial encontramos dos fallas de autenticación. CWE-916: las contraseñas se guardaban con SHA-256 y salt. El salt evita tablas arcoíris, pero SHA-256 sigue siendo rápido para probar claves en paralelo. CWE-307: el login no limitaba los intentos ni bloqueaba la cuenta.

### 01:55-02:25 · Fuerza bruta en vivo · diapositiva 4

**Pantalla:** ejecutar diez solicitudes fallidas desde el segundo terminal:

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

Si aparece `curl 000`, el servidor no respondió: comprobar que `main` está ejecutándose en el puerto 8080.

**Locución:**

> Enviamos diez contraseñas incorrectas al backend original. Las diez solicitudes reciben HTTP 401 Unauthorized en milisegundos. No aparece el 429 ni se bloquea la cuenta. Ese comportamiento permite automatizar fuerza bruta y credential stuffing.

### 02:25-03:00 · Benchmark · diapositiva 5

**Pantalla:** ejecutar `python3 scripts/benchmark_hash_cracking.py`; mostrar la medición local de CPU y la proyección GPU.

**Locución:**

> El benchmark mide cerca de medio microsegundo por hash para SHA-256 y 27 milisegundos para Argon2id en nuestro procesador. La tabla GPU es una proyección offline para claves alfanuméricas de ocho caracteres: una RTX 4090 supera 9.2 gigahashes por segundo con SHA-256, frente a unos 160 hashes por segundo con Argon2id. El tiempo medio proyectado pasa de 3.3 horas a más de 21 mil años. Braulio mostrará las defensas implementadas.

## Toma 3 · Braulio Robledo · 03:00-04:45 · diapositiva 6

**Pantalla:** abrir los archivos que correspondan en cada tramo. No se requiere servidor.

### 03:00-03:35 · Argon2id

Abrir `internal/auth/crypto.go` y mostrar los parámetros y `subtle.ConstantTimeCompare`.

**Locución:**

> Para mitigar CWE-916 migramos a Argon2id con 64 megabytes de memoria, tres iteraciones y cuatro hilos. La comparación en tiempo constante reduce filtraciones por diferencias de temporización.

### 03:35-04:10 · Rate limit y bloqueo

Mostrar `internal/auth/ratelimit.go` y `internal/store/user_store.go`, incluido Token Bucket, `Retry-After` y `RecordFailedLogin`.

**Locución:**

> CWE-307 se mitiga en dos capas. El Token Bucket limita el login a cinco solicitudes por minuto y responde 429 al exceder la cuota. Además, después de cinco fallos, `UserStore` bloquea la cuenta durante quince minutos y responde 423.

### 04:10-04:45 · Política de contraseñas y traspaso

Mostrar `internal/auth/password_policy.go` y `internal/auth/seclists_top500.txt`.

**Locución:**

> La política rechaza claves de menos de diez caracteres, las que contienen el nombre de usuario y las incluidas en la lista SecLists. El registro responde 400 si la contraseña no cumple. Es una política informada por NIST SP 800-63B; no afirmamos cumplimiento integral del estándar. Steven explicará MFA y mostrará las pruebas.

## Toma 4 · Jhon Steven · 04:45-06:30 · diapositivas 7-9

**Antes de grabar:** detener el servidor de `main`. Desde `taller_4`, iniciar el backend en una terminal y dejarlo activo; ejecutar la suite desde otra:

```bash
git switch taller_4
go run ./cmd/server
```

### 04:45-05:18 · MFA TOTP · diapositiva 7

**Pantalla:** abrir `internal/handlers/totp.go` y `internal/handlers/auth_handler.go`; mostrar `generateTOTPSetup`, `verifyTOTP` y `totp_code` para Admin.

**Locución:**

> Para proteger cuentas administrativas implementamos TOTP según RFC 6238. El usuario configura una semilla compatible con Google Authenticator o Aegis. El login de Admin requiere contraseña y el código vigente de seis dígitos; si falta o no es válido, se deniega el acceso con 401.

### 05:18-05:43 · Suite de seguridad · diapositiva 7

Ejecutar `bash test_auth_security.sh` en la segunda terminal y mostrar el resultado de las cuatro pruebas.

**Locución:**

> La suite cubre las cuatro defensas: 429 por exceso de solicitudes, 423 por bloqueo de cuenta, 400 ante contraseñas inseguras y 401 si Admin intenta entrar sin TOTP. Revisemos el resultado de cada prueba en la terminal.

### 05:43-06:03 · Mapeo CIS · diapositiva 8

**Locución:**

> CIS 5.2 trata de contraseñas únicas; nuestra política aporta controles, pero no demuestra unicidad y la alineación es parcial. CIS 6.3 aborda MFA en aplicaciones expuestas y nuestra cobertura es parcial porque TOTP protege solo a Admin. CIS 6.5 corresponde a MFA administrativa. El lockout mitiga CWE-307; no es CIS 6.3.

### 06:03-06:10 · Conclusiones · diapositiva 8

**Locución:**

> Un salt no vuelve lento un hash rápido. Argon2id encarece el cracking offline y las capas de acceso reducen los ataques automatizados.

### 06:10-06:30 · Autoría y uso responsable de IA · diapositiva 9

**Pantalla:** mostrar la declaración y la captura del encargo. Señalar ClickUp, la revisión por pares y la responsabilidad del equipo.

**Locución:**

> ClickUp organizó las doce tareas de este Taller 4 por responsable, prioridad y tres sprints. La guía de trabajo establece revisión por pares mediante Pull Request antes de integrar cambios. Usamos IA para apoyar la síntesis y el diseño; la responsabilidad técnica y académica sigue siendo del equipo. Muchas gracias.

## Ficha rápida

- **Juan José:** diapositivas 1-3; `/catalog` y estructura Go.
- **José Antonio Castelblanco:** diapositivas 4-5; código vulnerable, PoC y benchmark.
- **Braulio:** diapositiva 6; Argon2id, rate limit, lockout y política de contraseñas.
- **Steven:** diapositivas 7-9; TOTP, pruebas, CIS, conclusiones y declaración de IA.
