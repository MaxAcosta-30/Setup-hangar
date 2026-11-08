// Etapas del DeployWorkflow, deducidas de los mensajes que escribe el worker (deploy_activities.go).
export type Etapa = { id: string; nombre: string; detalle: string; inicio: string; fin: string }
export type EstadoEtapa = 'pendiente' | 'activo' | 'listo' | 'fallo'

export const ETAPAS: Etapa[] = [
  { id: 'clone', nombre: 'Clonar', detalle: 'git clone --depth=1', inicio: '[hangar] clonando', fin: 'repositorio clonado' },
  { id: 'build', nombre: 'Construir', detalle: 'docker build', inicio: 'iniciando docker build', fin: 'imagen construida' },
  { id: 'run', nombre: 'Ejecutar', detalle: 'red aislada + contenedor', inicio: 'creando red aislada', fin: 'iniciando contenedor' },
  { id: 'route', nombre: 'Enrutar', detalle: 'Traefik · subdominio', inicio: 'iniciando contenedor', fin: 'app disponible' },
]

export function estadoEtapas(mensajes: string[], estadoDespliegue: string): EstadoEtapa[] {
  const contiene = (texto: string) => mensajes.some((m) => m.includes(texto))
  let falloAsignado = false
  return ETAPAS.map((etapa) => {
    if (contiene(etapa.fin)) return 'listo'
    if (estadoDespliegue === 'failed' && !falloAsignado) {
      falloAsignado = true
      return 'fallo'
    }
    if (contiene(etapa.inicio) && estadoDespliegue !== 'failed') return 'activo'
    return 'pendiente'
  })
}
