# Marco Conceptual Técnico: OWASP Top 10 - A07:2021/2025 (Identification and Authentication Failures)

## 1. Evolución Histórica: A02:2017 a A07:2021/2025

* **OWASP Top 10 - 2017 (`A02: Broken Authentication`)**:
  * Alcance híbrido: agrupaba fallos de autenticación pura y gestión de sesiones (Session Fixation, IDs de sesión en URLs, exposición de tokens).
  * Posición #2: alta prevalencia debido a implementaciones monolíticas y personalizadas de login/sesión.
* **OWASP Top 10 - 2021/2025 (`A07: Identification and Authentication Failures`)**:
  * Descenso de #2 a #7: adopción generalizada de frameworks maduros, IdPs estandarizados (OAuth 2.0, OIDC, SAML) y servicios gestionados (Auth0, Cognito, Keycloak).
  * Redefinición conceptual: separación entre *identificación* (quién dice ser el sujeto) y *autenticación* (verificación de la prueba de identidad).
  * Enfoque actual: vectores modernos como bypass de MFA, abuso de credenciales a gran escala (Credential Stuffing), rotación deficiente de tokens JWT/API y desalineación con estándares modernos (NIST SP 800-63B).

---

## 2. Definición Formal, Vectores de Ataque e Impacto

* **Definición formal**: Defectos en el diseño, implementación o configuración de los mecanismos encargados de validar la identidad de un usuario, servicio o entidad, permitiendo la suplantación o evasión de controles de acceso.
* **Vectores de ataque habituales**:
  * *Credential Stuffing / Password Spraying*: uso de volcados de credenciales o contraseñas comunes sobre múltiples cuentas automatizadas.
  * *Brute-force distribuido*: ataques automatizados contra endpoints de login, reseteo de contraseña o códigos OTP/MFA.
  * *Session Hijacking / Fixation*: secuestro o fijación de identificadores de sesión no invalidados post-autenticación o transmitidos por canales no seguros.
  * *MFA Bypass*: explotación de lógica condicional (*fail-open*), endpoints desprotegidos que omiten el segundo factor o fatiga de notificaciones Push (*MFA Fatigue*).
  * *Broken Token Invalidation*: persistencia de tokens de acceso (JWT sin lista de revocación/expiración prolongada) tras el cierre de sesión.
* **Impacto técnico y de negocio**:
  * *Account Takeover (ATO)*: control total sobre cuentas de usuarios finales o administradores.
  * *Escalada de privilegios*: acceso vertical/horizontal no autorizado a datos sensibles.
  * *Exfiltración y manipulación de datos*: compromiso de confidencialidad e integridad del sistema.
  * *Incumplimiento normativo*: sanciones directas bajo GDPR, PCI-DSS, HIPAA o ISO 27001.

---

## 3. Desglose Detallado de CWEs Clave

### CWE-287: Improper Authentication
* **Mecanismo del fallo**: La aplicación no valida o valida incorrectamente la prueba de identidad de un actor antes de otorgarle acceso a un recurso.
* **Causas raíz**:
  * Lógica condicional deficiente (ej. validaciones dependientes del lado cliente, encabezados HTTP manipulables como `X-Forwarded-For` o flags en cookies).
  * Falta de firma o verificación criptográfica en tokens de identidad (ej. JWT con algoritmo `none` o clave pública usada como simétrica HMAC).
  * Rutas API internas o microservicios que asumen autenticación previa sin verificar el contexto.
* **Mitigación**:
  * Centralizar la autenticación en middlewares declarativos *fail-closed*.
  * Forzar validación criptográfica estricta de firmas y claims en tokens de sesión.
  * Tratar todas las entradas del cliente como no confiables.

### CWE-307: Improper Restriction of Excessive Authentication Attempts
* **Mecanismo del fallo**: Ausencia o insuficiencia de límites al número de intentos de autenticación fallidos consecutivos.
* **Causas raíz**:
  * Endpoints de autenticación (login, recuperación de contraseña, API de verificación OTP) sin *rate limiting*.
  * Controles basados únicamente en IP de origen, evadibles mediante redes de proxies o botnets.
  * Ausencia de detección de patrones anómalos distribuidos (*low and slow attacks*).
* **Mitigación**:
  * Implementar *rate limiting* multicapa: por IP, por cuenta objetivo y global.
  * Retardo progresivo (*exponential backoff*) y bloqueos temporales de cuentas con mecanismos de desbloqueo seguro.
  * Integración de CAPTCHA adaptativo o retos PoW (Proof of Work) tras *N* intentos fallidos.

### CWE-521: Weak Password Requirements
* **Mecanismo del fallo**: El sistema permite establecer secretos débiles, predecibles o conocidos, facilitando su adivinación u obtención offline.
* **Causas raíz**:
  * Reglas de longitud mínima insuficientes (< 8 caracteres).
  * Imposición de complejidad arbitraria (mayúsculas, números, símbolos) que genera patrones predecibles (ej. `Password1!`).
  * Falta de verificación contra diccionarios o listas de contraseñas comprometidas conocidas.
* **Mitigación**:
  * Alinear políticas con **NIST SP 800-63B**: longitud mínima de 12 a 15 caracteres.
  * Verificar contraseñas nuevas contra bases de datos de brechas (ej. HaveIBeenPwned API vía *k-Anonymity*).
  * Eliminar la expiración periódica obligatoria no justificada y las restricciones arbitrarias de caracteres.

### CWE-916: Use of Password Hash with Insufficient Computational Effort
* **Mecanismo del fallo**: Almacenamiento de credenciales utilizando funciones criptográficas rápidas o factores de coste computacional insuficientes, permitiendo ataques de fuerza bruta masivos offline vía GPU/ASIC.
* **Causas raíz**:
  * Uso de algoritmos de hashing de propósito general (MD5, SHA-1, SHA-256, SHA-512) diseñados para velocidad, no para almacenamiento de contraseñas.
  * Falta de sal (*salt*) criptográfica única por usuario, permitiendo ataques de tablas arcoíris (*rainbow tables*).
  * Factores de trabajo (*work factor* / iteraciones / memoria) desactualizados o estáticos.
* **Mitigación**:
  * Implementar funciones de derivación de claves adaptativas y resistentes a memoria:
    * **Argon2id**: memoria $\ge$ 64 MiB, iteraciones $\ge$ 3, paralelismo según hardware.
    * **bcrypt**: factor de coste $\ge$ 12.
    * **scrypt** o **PBKDF2-HMAC-SHA256** ($\ge$ 600,000 iteraciones según OWASP Cheat Sheet) como alternativas reguladas.
  * Generar siempre un *salt* criptográficamente seguro (mínimo 16 bytes) y único por usuario.
