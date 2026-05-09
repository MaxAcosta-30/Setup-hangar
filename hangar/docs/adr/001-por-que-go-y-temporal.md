# ADR 001 - Elección de Go y Temporal.io

**Fecha:** $(date +%Y-%m-%d)
**Estado:** Aceptado

## Contexto

El API Core necesita gestionar procesos del sistema operativo
(Docker Engine API), manejar concurrencia alta y ejecutar
workflows de deploy que deben sobrevivir reinicios del servidor.

## Decisión

- **Go** para el API Core y los Temporal Workers
- **Temporal.io** para orquestación de workflows de deploy
- **Fastify -> Fiber** (equivalente en Go)

## Por qué Go

- Binario único sin runtime externo - fácil de containerizar
- Manejo nativo de concurrencia con goroutines y channels
- SDK oficial de Docker Engine en Go
- Rendimiento predecible bajo carga de I/O intensivo

## Por qué Temporal sobre BullMQ o cron

- Los workflows de deploy duran 2-5 minutos
- Si el servidor se reinicia durante un build, Temporal
  retoma la ejecución desde la última Activity completada
- BullMQ pierde el estado si Redis se reinicia sin persistencia
- Cron no tiene concepto de estado, reintentos por step, ni timeouts

## Consecuencias positivas

- Deploys resilientes a fallos de infraestructura
- Visibilidad completa del estado de cada workflow en Temporal UI
- Workers pueden escalar horizontalmente sin coordinación

## Consecuencias a considerar

- Curva de aprendizaje inicial de Temporal (-1 día)
- Necesita un proceso worker separado del API server
- La restricción de determinismo en Workflows requiere disciplina
