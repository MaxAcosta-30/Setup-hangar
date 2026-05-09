# ADR 003 - Estrategia de aislamiento entre tenants

**Fecha:** 2026-05-09
**Estado:** Aceptado

## Contexto

Hangar ejecuta contenedores de múltiples usuarios en el mismo host Docker.
Sin aislamiento, App A podría hacer requests directos a App B usando su IP
interna de Docker, saltándose cualquier control de acceso a nivel de aplicación.

Necesitamos una estrategia que:
1. Impida comunicación lateral directa entre apps de distintos usuarios
2. Permita que Traefik alcance todos los contenedores para el routing
3. Permita que cada app acceda a internet normalmente (fetch, npm install, etc.)
4. Sea implementable sin Kubernetes ni herramientas externas

## Opciones consideradas

### Opción A: Una red compartida para todos
Todas las apps en la misma red Docker junto con Traefik.

**Ventajas:** Simple, cero overhead de configuración.
**Desventajas:** App A puede hacer `fetch("http://hangar-appB-id:3000")` directamente.
No hay aislamiento lateral. Inaceptable para un PaaS multiusuario.

### Opción B: Red interna por app (sin internet)
Cada app en su propia red con `internal: true`.

**Ventajas:** Aislamiento total.
**Desventajas:** Las apps no pueden hacer requests de salida (npm install en runtime,
APIs externas, webhooks). Inaceptable para apps de propósito general.

### Opción C: Red por app + conexión a red Traefik (elegida)
Cada app tiene su propia red bridge (`hangar-app-{id}`).
El contenedor se conecta a esa red Y a la red de Traefik (`infra_default`).

```
App A container
  |-- hangar-app-{idA}    <- red privada de A (solo A)
  \-- infra_default       <- red de Traefik (compartida con proxy)

App B container
  |-- hangar-app-{idB}    <- red privada de B (solo B)
  \-- infra_default       <- red de Traefik (compartida con proxy)

Traefik
  \-- infra_default       <- puede alcanzar A y B
```

**Resultado:**
- App A NO puede hacer requests a la IP interna de App B (redes separadas)
- Traefik SÍ puede alcanzar A y B (red compartida)
- A y B SÍ pueden hacer requests a internet (bridge sin `internal: true`)
- El único camino A→B es vía HTTP por sus subdominios públicos

## Decisión

Implementar la Opción C.

## Implementación

```go
// 1. Crear red aislada por app al momento del deploy
docker.NetworkCreate(ctx, "hangar-app-{appID}", types.NetworkCreate{
    Driver:   "bridge",
    Internal: false, // permite salida a internet
})

// 2. Crear contenedor conectado SOLO a la red de app
docker.ContainerCreate(ctx, config, hostConfig,
    &network.NetworkingConfig{
        EndpointsConfig: map[string]*network.EndpointSettings{
            "hangar-app-{appID}": {},
        },
    }, nil, containerName,
)

// 3. Post-start: conectar a red de Traefik
// (ContainerCreate solo acepta una red - NetworkConnect es la forma correcta)
docker.NetworkConnect(ctx, "infra_default", containerID, nil)
```

## Consecuencias positivas

- Aislamiento lateral sin herramientas externas
- Compatible con Docker Engine puro (no requiere K8s)
- Traefik detecta los contenedores correctamente vía labels
- Las apps mantienen acceso completo a internet

## Consecuencias a considerar

- Cada app ocupa una red Docker adicional (overhead mínimo: ~1KB de metadata)
- `CleanupApp` debe eliminar la red al borrar la app para evitar acumulación
- El nombre de red debe ser determinista: `hangar-app-{appID}` -
  centralizado en `appNetworkName()` para evitar inconsistencias

## Verificación

```bash
# Ver redes aisladas de cada app
docker network ls | grep hangar-app

# Confirmar que App A no puede alcanzar App B directamente
docker exec hangar-{appA-id} wget -T2 http://$(docker inspect hangar-{appB-id} \
  --format '{{.NetworkSettings.Networks.hangar-app-{appB-id}.IPAddress}}'):3000
# -> debe fallar con timeout (no hay ruta entre redes)

# Confirmar que App A sí puede salir a internet
docker exec hangar-{appA-id} wget -T5 -q -O- https://ifconfig.me
# -> debe devolver la IP pública del host
```

