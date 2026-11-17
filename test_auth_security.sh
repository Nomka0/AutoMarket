#!/usr/bin/env bash
set -eo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
RUN_ID=$(date +%s)

# Colores de salida
GREEN="\033[1;32m"
RED="\033[1;31m"
BLUE="\033[1;34m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RESET="\033[0m"

TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

log_header() {
  echo -e "\n${BLUE}======================================================================${RESET}"
  echo -e "${CYAN}  $1${RESET}"
  echo -e "${BLUE}======================================================================${RESET}"
}

report_result() {
  local test_name="$1"
  local expected="$2"
  local actual="$3"
  local details="$4"
  
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  if [ "$expected" -eq "$actual" ]; then
    PASSED_TESTS=$((PASSED_TESTS + 1))
    echo -e "  ${GREEN}[PASS]${RESET} $test_name | HTTP $actual (Esperado: $expected) $details"
  else
    FAILED_TESTS=$((FAILED_TESTS + 1))
    echo -e "  ${RED}[FAIL]${RESET} $test_name | Obtenido: HTTP $actual (Esperado: $expected) $details"
  fi
}

# 0. Verificación inicial de conectividad
echo -e "${CYAN}Iniciando Suite de Pruebas Automatizadas de Seguridad (OWASP A07)${RESET}"
echo -e "Target: ${YELLOW}$BASE_URL${RESET}"

SERVER_UP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/ping" || echo "000")
if [ "$SERVER_UP" -ne 200 ]; then
  echo -e "${RED}[ERROR] No se pudo conectar a $BASE_URL/ping (HTTP $SERVER_UP). Inicie el servidor primero.${RESET}"
  exit 1
fi

# ==============================================================================
# Prueba 1: Rate Limiting (CWE-307)
# 8 peticiones consecutivas a /api/auth/login. A partir de la 6ta debe responder 429.
# ==============================================================================
log_header "Prueba 1: Rate Limiting por IP (CWE-307) - Max 5 req/min"
RATE_LIMIT_IP="198.51.100.$((RUN_ID % 200 + 10))"
p1_success=true

for i in $(seq 1 8); do
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: $RATE_LIMIT_IP" \
    -d "{\"username\":\"rate_user_$RUN_ID\",\"password\":\"WrongPassword#2026\"}")
  
  if [ "$i" -le 5 ]; then
    # Primeras 5 peticiones permitidas (rechazadas por credenciales -> 401)
    if [ "$HTTP_CODE" -ne 401 ]; then
      p1_success=false
      echo -e "    Req #$i -> HTTP $HTTP_CODE (esperaba 401)"
    fi
  else
    # A partir de la 6ta peticion debe bloquear por cuota -> 429
    if [ "$HTTP_CODE" -ne 429 ]; then
      p1_success=false
      echo -e "    Req #$i -> HTTP $HTTP_CODE (esperaba 429)"
    fi
  fi
done

if [ "$p1_success" = true ]; then
  report_result "Rate Limiting en /api/auth/login (Req 1-5: 401, Req 6-8: 429)" 429 429 "(Req 6 a 8 bloqueadas con Too Many Requests)"
else
  report_result "Rate Limiting en /api/auth/login (Req 1-5: 401, Req 6-8: 429)" 429 000 "(Alguna petición no retornó el código esperado)"
fi

# ==============================================================================
# Prueba 2: Account Lockout (CIS Control 6 / CWE-307)
# 5 intentos con clave errónea para un usuario. El 6to debe responder HTTP 423 Locked.
# ==============================================================================
log_header "Prueba 2: Account Lockout tras 5 intentos fallidos (HTTP 423)"
LOCKOUT_USER="victim_lockout_${RUN_ID}"
LOCKOUT_PASS="SuperSegura#2026"

# 2.1 Registrar usuario legítimo
REG_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"$LOCKOUT_PASS\",\"contact_info\":\"lockout@correo.com\"}")

if [ "$REG_CODE" -ne 201 ]; then
  echo -e "  ${RED}[WARN] No se pudo crear usuario para prueba de lockout (HTTP $REG_CODE)${RESET}"
fi

# 2.2 Enviar 5 intentos fallidos (usando IPs distintas para aislar del rate limiting)
p2_attempts_ok=true
for i in $(seq 1 5); do
  ATTEMPT_IP="203.0.113.$((i + 10))"
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: $ATTEMPT_IP" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"ClaveEquivocada#$i\"}")
  if [ "$CODE" -ne 401 ]; then
    p2_attempts_ok=false
    echo -e "    Intento fallido #$i -> HTTP $CODE (esperaba 401)"
  fi
done

# 2.3 6to intento: debe responder HTTP 423 Locked
CHECK_IP="203.0.113.99"
LOCKOUT_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: $CHECK_IP" \
  -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"$LOCKOUT_PASS\"}")

report_result "Account Lockout en 6to intento para usuario bloqueado" 423 "$LOCKOUT_HTTP_CODE" "(Cuenta bloqueada temporalmente por 15 min)"

# ==============================================================================
# Prueba 3: Política de Contraseñas (NIST SP 800-63B / CIS 5.2 / CWE-521)
# Intento de registro con contraseña débil '123456'. Debe responder HTTP 400.
# ==============================================================================
log_header "Prueba 3: Política de Contraseñas NIST 800-63B (HTTP 400)"
WEAK_USER="weak_tester_${RUN_ID}"
WEAK_PASS="123456"

PWD_POLICY_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$WEAK_USER\",\"password\":\"$WEAK_PASS\",\"contact_info\":\"weak@correo.com\"}")

report_result "Rechazo de contraseña corta/débil ('$WEAK_PASS' < 10 caracteres)" 400 "$PWD_POLICY_HTTP_CODE" "(Bad Request por incumplimiento de política)"

# ==============================================================================
# Prueba 4: MFA TOTP para Cuentas Privilegiadas (CIS Control 6.5)
# Intento de login de administrador sin código TOTP. Debe responder HTTP 401.
# ==============================================================================
log_header "Prueba 4: Autenticación Multifactor (MFA TOTP) para Admin (HTTP 401)"
ADMIN_USER="${ADMIN_USERNAME:-admin}"
ADMIN_PASS="${ADMIN_PASSWORD:-ciberinsegurid4d#}"

# Verificar si admin existe; si no, crear con setup-admin
HAS_ADMIN=$(curl -s "$BASE_URL/api/auth/has-admin" | grep -o '"has_admin":true' || true)
if [ -z "$HAS_ADMIN" ]; then
  curl -s -o /dev/null -X POST "$BASE_URL/api/auth/setup-admin" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}"
fi

# Intento de login como admin con password válida pero SIN código TOTP
MFA_IP="198.51.100.88"
MFA_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: $MFA_IP" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")

report_result "Exigencia de segundo factor TOTP para cuenta Admin" 401 "$MFA_HTTP_CODE" "(Login denegado por ausencia de código TOTP RFC 6238)"

# ==============================================================================
# Resumen Final de Ejecución
# ==============================================================================
echo -e "\n${BLUE}======================================================================${RESET}"
echo -e "${CYAN}  RESUMEN DE PRUEBAS DE SEGURIDAD (TALLER 4 - OWASP A07)${RESET}"
echo -e "${BLUE}======================================================================${RESET}"
echo -e "  Total de Pruebas : ${TOTAL_TESTS}"
echo -e "  Aprobadas        : ${GREEN}${PASSED_TESTS}${RESET}"
echo -e "  Fallidas         : ${RED}${FAILED_TESTS}${RESET}"

if [ "$FAILED_TESTS" -eq 0 ]; then
  echo -e "\n${GREEN}✔ TODAS LAS PRUEBAS DE SEGURIDAD FUERON SUPERADAS EXITOSAMENTE [PASS]${RESET}\n"
  exit 0
else
  echo -e "\n${RED}✘ SE ENCONTRARON FALLAS EN LA VERIFICACIÓN DE SEGURIDAD [FAIL]${RESET}\n"
  exit 1
fi
