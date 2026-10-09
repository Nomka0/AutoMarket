# Taller en Equipo

## Implementación de AAA en sitio web

**Ciberseguridad**

Iván M. Cabezas T,

---

## Contexto - AutoMarket

• Sitio web para el despliegue de información sobre venta de vehículos usados. La plataforma expone un catálogo público de publicaciones y ofrece funcionalidades diferenciadas según el tipo de usuario que la consuma.  
• Su equipo actuará como **desarrollador de un componente backend** del sistema.  
• El docente actuará como **líder técnico y revisor del VIDEO** asociado al Pull Request que ustedes entregarán.

---

## Roles del sistema

• Descripción de los roles del sitema

| Rol               | Descripción                                                                                           |
| :---------------- | :---------------------------------------------------------------------------------------------------- |
| **Visitante**     | Persona no autenticada que consulta el catálogo público y los datos de contacto de los vendedores     |
| **Vendedor**      | Persona registrada y autenticada que publica sus vehículos en venta y reporta sus ventas.             |
| **Administrador** | Persona autenticada con privilegios para autorizar publicaciones y eliminar publicaciones o usuarios. |

---

## Historia de Usuario · Visitante

• **HU-VIS-01 · Consulta pública del catálogo y de la información de contacto**

**Como** visitante del sitio,  
**quiero** consultar el catálogo de vehículos en venta y acceder a la información de contacto del vendedor,  
**para** evaluar la oferta y comunicarme directamente con quien ofrece un vehículo de mi interés, sin necesidad de registrarme.

**Criterio de Aceptación :**  
**Dado** que soy un visitante no autenticado,  
**cuando** consulto un vehículo del catálogo público,  
**entonces** el sistema me responde con la información del vehículo y con los datos de contacto del vendedor asociado.

---

## Historia de Usuario · Vendedor

• **HU-VEN-01 · Publicación y gestión autónoma de vehículos en venta**

**Como** vendedor,  
**quiero** registrarme autónomamente en la plataforma, publicar mis vehículos en venta y reportar cuando los venda,  
**para** ofrecer mis vehículos en el catálogo y retirarlos cuando concrete la operación.

**Criterio de Aceptación :**  
**Dado** que soy un vendedor registrado y autenticado,  
**cuando** envío la solicitud de publicación de un vehículo con datos válidos,  
**entonces** el sistema registra la publicación en estado pendiente de aprobación asociándola a mi cuenta.

---

## Historia de Usuario · Administrador

• **HU-ADM-01 · Moderación de publicaciones y usuario**

**Como** administrador del sitio,  
**quiero** autorizar las publicaciones enviadas por los vendedores y eliminar publicaciones o usuarios cuando sea necesario,  
**para** garantizar la calidad e integridad del catálogo y de la comunidad de la plataforma..

**Criterio de Aceptación :**  
**Dado** que soy un administrador autenticado,  
**cuando** autorizo una publicación en estado pendiente de aprobación,  
**entonces** el sistema cambia su estado a publicada y la hace visible en el catálogo público.

---

## Requerimiento Funcional

• **RF-01 · Marco de seguridad AAA**  
El sistema deberá implementar el marco de seguridad AAA (_Authentication, Authorization, Accounting_) sobre todas las operaciones expuestas por la API.

### Authentication

• Cuando un solicitante intente ejecutar una operación sobre un recurso protegido, el sistema deberá autenticar su identidad antes de procesar la solicitud.

### Authorization

• Cuando un solicitante autenticado invoque una operación, el sistema deberá autorizarla conforme al rol asignado al solicitante.

### Accounting

• Cuando el sistema ejecute cualquier operación sobre un recurso, deberá registrarla de forma trazable e íntegra.

---

## Restricciones técnicas

• **Lenguaje:** Go (solo standard library).  
• **Framework web:** Gin (`github.com/gin-gonic/gin`) como **única dependencia externa permitida**.  
• **Protocolo:** HTTP (sin TLS/HTTPS en esta entrega).  
• **Persistencia:** libre elección del equipo, siempre dentro de las restricciones anteriores.  
• **Prohibido:** cualquier librería externa adicional (JWT, bcrypt, Casbin, Zap, ORMs, drivers de BD, entre otros).

---

## Alcance y Entrega

### ALCANCE

• La solución debe ceñirse **estrictamente** a lo pedido en las HUs y en el RF-01.  
• Se penalizarán funcionalidades, endpoints, mecanismos o capacidades **no solicitadas** en los artefactos entregados, aunque el equipo las considere "buenas prácticas".  
• La calidad inherente al código (tales como separación de responsabilidades, cohesión, bajo acoplamiento, manejo consistente de errores y ausencia de duplicación, entre otros), y el entendimiento del mismo, durante el video, o en sesiones de sustentación en vivo, será factor de evaluación.

### ENTREGA

• Un único archivo PDF que incluya: portada, integrantes (Apellido – Nombre, en orden alfabético), URL de repositorio y hash del commit, invitación al Usuario **imcabezas**, URL del video de demostración/sustentación, un (1) diagrama de estructura y un (1) diagrama de comportamiento de la solución, declaración de uso responsable de IA.
