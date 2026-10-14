#!/usr/bin/env bash
set -e
BASE_URL="http://localhost:8080"

# Colores para salida clara en consola
GREEN="\033[32m"
RED="\033[31m"
BLUE="\033[34m"
YELLOW="\033[33m"
RESET="\033[0m"

step() {
  echo -e "\n${BLUE}==>${RESET} $1"
}

assert_status() {
  local expected="$1"
  local actual="$2"
  local desc="$3"
  if [ "$expected" -eq "$actual" ]; then
    echo -e "  ${GREEN}[PASS]${RESET} HTTP $actual | $desc"
  else
    echo -e "  ${RED}[FAIL]${RESET} Se esperaba HTTP $expected pero se obtuvo HTTP $actual | $desc"
    exit 1
  fi
}

json_get() {
  local key="$1"
  if command -v jq >/dev/null 2>&1; then
    jq -r ".$key // empty"
  else
    python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('$key', ''))"
  fi
}

json_has() {
  local match="$1"
  if grep -q "$match"; then
    return 0
  fi
  return 1
}

# 0. Verificar conectividad con el servidor
step "0. Verificando que el servidor esté activo"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/ping" || echo "000")
if [ "$HTTP_CODE" -ne 200 ]; then
  echo -e "${RED}[ERROR]${RESET} No se puede conectar a $BASE_URL/ping (HTTP $HTTP_CODE)."
  echo "Asegúrate de iniciar el servidor en otra terminal antes de ejecutar este script:"
  echo "  go run ./cmd/server"
  exit 1
fi
assert_status 200 "$HTTP_CODE" "GET /ping servidor respondiendo correctamente"

# Generar sufijo único para evitar colisiones en ejecuciones repetidas
RUN_ID=$(date +%s)
SELLER1_USER="vendedor1_${RUN_ID}"
SELLER2_USER="vendedor2_${RUN_ID}"
ADMIN_USER="${ADMIN_USERNAME:-admin}"
ADMIN_PASS="${ADMIN_PASSWORD:-ciberinsegurid4d#}"

# 1. Visitante: Catálogo público inicial
step "1. HU-VIS-01: Visitante consulta catálogo público inicial"
HTTP_CODE=$(curl -s -o /tmp/catalog.json -w "%{http_code}" "$BASE_URL/api/catalog")
assert_status 200 "$HTTP_CODE" "GET /api/catalog accesible sin autenticación"

# 2. Autenticación: Registro de usuarios
step "2. RF-01 / HU-VEN-01: Registro de vendedores"
# 2.1 Registro exitoso vendedor 1
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER1_USER}\",\"password\":\"pass1234\",\"contact_info\":\"juan@correo.com / 3001112233\"}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 201 "$HTTP_CODE" "POST /api/auth/register vendedor 1 registrado"
SELLER1_ID=$(echo "$BODY" | json_get "id")

# 2.2 Registro exitoso vendedor 2
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER2_USER}\",\"password\":\"pass1234\",\"contact_info\":\"maria@correo.com / 3114445566\"}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 201 "$HTTP_CODE" "POST /api/auth/register vendedor 2 registrado"
SELLER2_ID=$(echo "$BODY" | json_get "id")

# 2.3 Intento de registro duplicado -> 400 Bad Request
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER1_USER}\",\"password\":\"pass1234\",\"contact_info\":\"otro@correo.com\"}")
assert_status 400 "$HTTP_CODE" "POST /api/auth/register rechaza nombre de usuario duplicado"

# 3. Autenticación: Inicio de sesión
step "3. RF-01 Authentication: Inicio de sesión y emisión de tokens Bearer"
# 3.1 Login Vendedor 1
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER1_USER}\",\"password\":\"pass1234\"}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "POST /api/auth/login vendedor 1 autenticado"
TOKEN_SELLER1=$(echo "$BODY" | json_get "token")

# 3.2 Login Vendedor 2
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER2_USER}\",\"password\":\"pass1234\"}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "POST /api/auth/login vendedor 2 autenticado"
TOKEN_SELLER2=$(echo "$BODY" | json_get "token")

# 3.3 Login Administrador
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "POST /api/auth/login administrador autenticado"
TOKEN_ADMIN=$(echo "$BODY" | json_get "token")

# 3.4 Intento de login con credenciales erróneas -> 401 Unauthorized
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${SELLER1_USER}\",\"password\":\"clave_incorrecta\"}")
assert_status 401 "$HTTP_CODE" "POST /api/auth/login rechaza contraseña incorrecta con 401"

# 4. Publicación de vehículo (Vendedor) y controles de autorización
step "4. HU-VEN-01 / RF-01: Publicación de vehículo y controles AAA"
# 4.1 Intento de publicar sin token -> 401 Unauthorized
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/seller/vehicles" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Mazda 3\",\"description\":\"Excelente estado\",\"price\":45000000,\"year\":2020}")
assert_status 401 "$HTTP_CODE" "POST /api/seller/vehicles rechaza petición sin token"

# 4.2 Intento de publicar con rol admin (no vendedor) -> 403 Forbidden
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/seller/vehicles" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Mazda 3\",\"description\":\"Excelente estado\",\"price\":45000000,\"year\":2020}")
assert_status 403 "$HTTP_CODE" "POST /api/seller/vehicles deniega rol admin con 403"

# 4.3 Intento de publicar datos con campos vacíos o solo espacios -> 400 Bad Request
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/seller/vehicles" \
  -H "Authorization: Bearer ${TOKEN_SELLER1}" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"   \",\"description\":\"Solo espacios\",\"price\":45000000,\"year\":2020}")
assert_status 400 "$HTTP_CODE" "POST /api/seller/vehicles rechaza título con espacios en blanco con 400"

# 4.4 Vendedor 1 publica vehículo válido -> 201 Created (estado PENDING)
RES=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/seller/vehicles" \
  -H "Authorization: Bearer ${TOKEN_SELLER1}" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Mazda 3 Touring\",\"description\":\"Unico dueno, 45000 km\",\"price\":52000000,\"year\":2021}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 201 "$HTTP_CODE" "POST /api/seller/vehicles registra vehículo en estado PENDING"
VEHICLE_ID=$(echo "$BODY" | json_get "id")
VEHICLE_STATUS=$(echo "$BODY" | json_get "status")
if [ "$VEHICLE_STATUS" != "PENDING" ]; then
  echo -e "  ${RED}[FAIL]${RESET} El estado inicial debe ser PENDING, se obtuvo: $VEHICLE_STATUS"
  exit 1
fi
echo -e "  ${GREEN}[INFO]${RESET} ID generado: $VEHICLE_ID | Estado: $VEHICLE_STATUS"

# 5. Moderación Admin y Visibilidad en Catálogo
step "5. HU-ADM-01 / HU-VIS-01: Moderación de publicaciones"
# 5.1 Intento de listar pendientes sin token -> 401
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/admin/publications/pending")
assert_status 401 "$HTTP_CODE" "GET /api/admin/publications/pending rechaza petición sin token"

# 5.2 Intento de listar pendientes con rol vendedor -> 403
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/admin/publications/pending" \
  -H "Authorization: Bearer ${TOKEN_SELLER1}")
assert_status 403 "$HTTP_CODE" "GET /api/admin/publications/pending deniega rol vendedor con 403"

# 5.3 Admin lista pendientes -> 200 OK
RES=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/admin/publications/pending" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "GET /api/admin/publications/pending lista pendientes para admin"
if ! echo "$BODY" | json_has "$VEHICLE_ID"; then
  echo -e "  ${RED}[FAIL]${RESET} El vehículo $VEHICLE_ID no aparece en la lista de pendientes"
  exit 1
fi

# 5.4 Visitante NO debe ver el vehículo PENDING en el catálogo público
RES=$(curl -s "$BASE_URL/api/catalog")
if echo "$RES" | json_has "$VEHICLE_ID"; then
  echo -e "  ${RED}[FAIL]${RESET} Vehículo en estado PENDING es visible en el catálogo público"
  exit 1
fi
echo -e "  ${GREEN}[PASS]${RESET} Vehículo PENDING no es visible en el catálogo público"

# 5.5 Visitante consulta detalle del vehículo PENDING -> 404 Not Found
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/catalog/${VEHICLE_ID}")
assert_status 404 "$HTTP_CODE" "GET /api/catalog/:id retorna 404 para publicación PENDING"

# 5.6 Admin aprueba la publicación -> 200 OK (estado APPROVED)
RES=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/api/admin/publications/${VEHICLE_ID}/approve" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "PATCH /api/admin/publications/:id/approve aprueba publicación"
APPROVED_STATUS=$(echo "$BODY" | json_get "status")
if [ "$APPROVED_STATUS" != "APPROVED" ]; then
  echo -e "  ${RED}[FAIL]${RESET} El estado tras aprobación debe ser APPROVED, se obtuvo: $APPROVED_STATUS"
  exit 1
fi

# 5.7 Intento de aprobar nuevamente -> 409 Conflict
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/api/admin/publications/${VEHICLE_ID}/approve" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}")
assert_status 409 "$HTTP_CODE" "PATCH /api/admin/publications/:id/approve rechaza doble aprobación con 409"

# 6. Catálogo Público con Publicación Aprobada
step "6. HU-VIS-01: Verificación de publicación aprobada en catálogo"
# 6.1 Catálogo público contiene el vehículo
RES=$(curl -s "$BASE_URL/api/catalog")
if ! echo "$RES" | json_has "$VEHICLE_ID"; then
  echo -e "  ${RED}[FAIL]${RESET} El vehículo APPROVED $VEHICLE_ID no aparece en el catálogo"
  exit 1
fi
echo -e "  ${GREEN}[PASS]${RESET} Vehículo APPROVED visible en catálogo público"

# 6.2 Detalle del vehículo incluye datos y contacto del vendedor
RES=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/catalog/${VEHICLE_ID}")
HTTP_CODE=$(echo "$RES" | tail -n1)
BODY=$(echo "$RES" | sed '$d')
assert_status 200 "$HTTP_CODE" "GET /api/catalog/:id retorna detalle del vehículo aprobado"
if ! echo "$BODY" | json_has "juan@correo.com"; then
  echo -e "  ${RED}[FAIL]${RESET} El detalle no expone los datos de contacto del vendedor (seller_contact)"
  exit 1
fi
echo -e "  ${GREEN}[PASS]${RESET} seller_contact expuesto correctamente al visitante"

# 7. Reportar Vendido (Control de Acceso Horizontal)
step "7. HU-VEN-01 / RF-01: Reportar vehículo vendido y control de propiedad"
# 7.1 Vendedor 2 intenta marcar como vendido el vehículo de Vendedor 1 -> 403 Forbidden
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/api/seller/vehicles/${VEHICLE_ID}/sold" \
  -H "Authorization: Bearer ${TOKEN_SELLER2}")
assert_status 403 "$HTTP_CODE" "PATCH /api/seller/vehicles/:id/sold rechaza modificación por otro vendedor con 403"

# 7.2 Vendedor 1 (dueño) marca como vendido su vehículo -> 200 OK
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/api/seller/vehicles/${VEHICLE_ID}/sold" \
  -H "Authorization: Bearer ${TOKEN_SELLER1}")
assert_status 200 "$HTTP_CODE" "PATCH /api/seller/vehicles/:id/sold dueño marca como vendido con 200"

# 7.3 Intento de marcar nuevamente como vendido -> 409 Conflict
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/api/seller/vehicles/${VEHICLE_ID}/sold" \
  -H "Authorization: Bearer ${TOKEN_SELLER1}")
assert_status 409 "$HTTP_CODE" "PATCH /api/seller/vehicles/:id/sold rechaza marcar vendido nuevamente con 409"

# 7.4 Vehículo vendido ya NO debe ser visible en el catálogo público
RES=$(curl -s "$BASE_URL/api/catalog")
if echo "$RES" | json_has "$VEHICLE_ID"; then
  echo -e "  ${RED}[FAIL]${RESET} Vehículo en estado SOLD sigue visible en el catálogo público"
  exit 1
fi
echo -e "  ${GREEN}[PASS]${RESET} Vehículo SOLD retirado exitosamente del catálogo público"

# 7.5 Visitante consulta detalle de vendido -> 404 Not Found
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/catalog/${VEHICLE_ID}")
assert_status 404 "$HTTP_CODE" "GET /api/catalog/:id retorna 404 para vehículo SOLD"

# 8. Eliminación de Usuarios e Invalidación de Sesiones
step "8. HU-ADM-01 / RF-01: Eliminación de usuarios e invalidación de sesiones"
# 8.1 Admin intenta autoeliminarse -> 400 Bad Request
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/admin/users/admin-1" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}")
assert_status 400 "$HTTP_CODE" "DELETE /api/admin/users/:id rechaza autoeliminación de admin con 400"

# 8.2 Admin elimina a Vendedor 2 -> 200 OK
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/admin/users/${SELLER2_ID}" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}")
assert_status 200 "$HTTP_CODE" "DELETE /api/admin/users/:id elimina usuario con 200"

# 8.3 Vendedor 2 intenta operar con su token previo -> 401 Unauthorized (sesión revocada)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/seller/vehicles" \
  -H "Authorization: Bearer ${TOKEN_SELLER2}" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Renault Clio\",\"description\":\"Buen auto\",\"price\":20000000,\"year\":2018}")
assert_status 401 "$HTTP_CODE" "Token de usuario eliminado revocado automáticamente con 401"

# 9. Verificación de Accounting (RF-01)
step "9. RF-01 Accounting: Verificación de trazabilidad en audit.log"
if [ ! -f "audit.log" ]; then
  echo -e "  ${RED}[FAIL]${RESET} Archivo audit.log no encontrado"
  exit 1
fi

TOTAL_LOGS=$(wc -l < audit.log)
if [ "$TOTAL_LOGS" -lt 5 ]; then
  echo -e "  ${RED}[FAIL]${RESET} audit.log contiene muy pocos registros ($TOTAL_LOGS)"
  exit 1
fi
echo -e "  ${GREEN}[PASS]${RESET} audit.log generado y acumulando registros (Total: $TOTAL_LOGS líneas)"

# Validar que audit.log tiene campos obligatorios de RF-01
LAST_LOG=$(tail -n1 audit.log)
for campo in "timestamp" "method" "path" "user_id" "role" "status_code" "duration_ms"; do
  if ! echo "$LAST_LOG" | grep -q "$campo"; then
    echo -e "  ${RED}[FAIL]${RESET} audit.log carece del campo requerido: $campo"
    exit 1
  fi
done
echo -e "  ${GREEN}[PASS]${RESET} Estructura de auditoría RFC-conforme verificada en audit.log"

echo -e "\n${GREEN}======================================================"
echo "  TODAS LAS PRUEBAS E2E PASARON EXITOSAMENTE (US-13)"
echo -e "======================================================${RESET}\n"
