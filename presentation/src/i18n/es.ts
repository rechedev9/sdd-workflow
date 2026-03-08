import type { Translations } from './en';

export const es: Translations = {
  nav: { prev: 'Anterior', next: 'Siguiente' },
  hero: {
    hookLine: '147.822 líneas de TypeScript. 13 días. Un solo desarrollador.',
    hookSub: 'Esto es lo que pasa cuando dejas de hacer prompts y empiezas a orquestar.',
    title: 'SHENRON',
    subtitle: 'Framework de Desarrollo Guiado por Especificaciones',
    tagline: 'Reúne las specs. Invoca el código.',
    version: 'v1.1',
    versionLabel: 'Razonamiento Semi-Formal',
  },
  problem: {
    title: 'El Problema',
    subtitle: 'Seis modos de fallo de la programación asistida por IA',
    items: [
      {
        title: 'Contaminación de Contexto',
        description: 'Un agente lee todo el código. La atención se diluye — se editan los archivos equivocados.',
      },
      {
        title: 'Amnesia de Sesión',
        description: 'Cada sesión parte de cero. Decisiones, bugs resueltos, convenciones — todo olvidado.',
      },
      {
        title: 'Sin Trazabilidad',
        description: 'El código se entrega sin registro del porqué. Meses después, nadie sabe.',
      },
      {
        title: 'Alucinaciones',
        description: 'Contexto lleno fuerza compresión. Las restricciones desaparecen; aparece lógica inventada.',
      },
      {
        title: 'Sin Verificación',
        description: 'La IA revisa su propio código — mismo autor, mismos puntos ciegos.',
      },
      {
        title: 'Versión Incorrecta',
        description: 'Entrenada con docs viejos, sugiere patrones obsoletos para tu stack actual.',
      },
    ],
  },
  insight: {
    quote: 'No necesitas un modelo más inteligente.',
    quoteLine2: 'Necesitas un flujo de trabajo más inteligente.',
    description:
      'El mismo modelo produce resultados drásticamente mejores cuando recibe un contexto limpio con instrucciones enfocadas y el material de referencia correcto. El problema no es la inteligencia del modelo — es la arquitectura del flujo de trabajo: cómo se estructura la interacción entre el humano, el modelo y el codebase.',
  },
  architecture: {
    title: 'Arquitectura',
    subtitle: 'Cómo encajan las piezas',
    orchestrator: {
      title: 'El Coordinador',
      description:
        'El coordinador del proyecto. Asigna tareas, rastrea el progreso y pide tu aprobación en cada paso. Nunca escribe código — delega todo.',
    },
    subAgents: {
      title: 'Trabajadores Especializados',
      description:
        'Un asistente de IA nuevo para cada fase, con solo la información que necesita. Sin distracciones, sin sobrecarga de información.',
    },
    artifacts: {
      title: 'Documentación Viva',
      description:
        'Cada decisión, plan y revisión guardado como archivos. Nada se pierde ni se recuerda mal — el proyecto se documenta a sí mismo.',
    },
    memory: {
      title: 'Memoria del Proyecto',
      description:
        'Una base de datos que recuerda decisiones, bugs resueltos y patrones de todas las sesiones. Como un desarrollador senior que nunca olvida.',
    },
    skills: {
      title: 'Guías de Conocimiento',
      description:
        'Archivos de conocimiento actualizados para cada tecnología. Garantiza que la IA use patrones de React 19 para React 19 — nunca respuestas desactualizadas.',
    },
  },
  pillars: {
    title: 'Los Cinco Pilares',
    subtitle: 'Cada pilar resuelve un modo de fallo específico de la programación con IA',
    items: [
      {
        title: 'Divide y Especializa',
        description:
          'Cada paso lo maneja una IA nueva con solo lo que necesita. Como contratar un especialista para cada tarea en vez de un generalista sobrecargado.',
        detail:
          'El agente de implementación nunca ve todo el codebase — solo el plan de diseño y el archivo específico que está modificando. Contexto enfocado produce resultados enfocados.',
      },
      {
        title: 'Engram Memory',
        description:
          'Un sistema de memoria que persiste decisiones, bugs y patrones entre sesiones. Empieza un día nuevo y retoma exactamente donde lo dejaste.',
        detail:
          'Al inicio de sesión, las decisiones previas se cargan automáticamente. Cada nueva decisión, corrección de bug y descubrimiento se guarda de inmediato — no al final de la sesión.',
      },
      {
        title: 'Libro de Reglas de Revisión',
        description:
          'Un conjunto de reglas escritas que un revisor independiente verifica en cada cambio. El revisor nunca escribió el código que revisa — sin sesgos, sin puntos ciegos.',
        detail:
          'Las reglas usan palabras clave claras: REJECT (bloqueo duro), REQUIRE (debe justificarse), PREFER (informativo). Cada regla está versionada y es transparente.',
      },
      {
        title: 'Experiencia Actualizada',
        description:
          'Guías de conocimiento que se actualizan solas para cada tecnología. La IA sabe qué versión de cada herramienta usas y aplica los patrones correctos.',
        detail:
          'Si una guía no tiene la respuesta, busca en internet y se actualiza. Las brechas de conocimiento se cierran con el tiempo.',
      },
      {
        title: 'Razonamiento Semi-Formal',
        description:
          'Obliga a la IA a razonar paso a paso antes de actuar. Como pedirle a un cirujano que escriba un plan antes del primer corte.',
        detail:
          'Cuatro protocolos de pensamiento estructurado aplicados en las fases más críticas: investigación, implementación, revisión y verificación.',
        isV11: true,
      },
    ],
  },
  pipeline: {
    title: 'El Pipeline',
    subtitle: '11 fases desde init hasta archive',
    phases: [
      {
        name: 'init',
        description: 'Configura el proyecto y detecta el stack tecnológico.',
        v11: false,
      },
      {
        name: 'explore',
        description: 'Investiga el codebase — entiende lo que existe antes de cambiar cualquier cosa.',
        v11: true,
      },
      {
        name: 'propose',
        description: 'Escribe una propuesta en lenguaje claro: qué cambia, por qué, y cómo revertirlo si es necesario.',
        v11: false,
      },
      {
        name: 'spec',
        description: 'Define requisitos formales: exactamente qué debe hacer el código, con criterios de aceptación.',
        v11: false,
      },
      {
        name: 'design',
        description: 'Planifica la arquitectura técnica antes de escribir una sola línea de código.',
        v11: false,
      },
      {
        name: 'tasks',
        description: 'Divide el diseño en una lista de implementación numerada y organizada por fases.',
        v11: false,
      },
      {
        name: 'apply',
        description: 'Escribe el código tarea por tarea, con verificación de build automática después de cada lote.',
        v11: true,
      },
      {
        name: 'review',
        description: 'Revisión de código independiente contra requisitos, reglas y el diseño original.',
        v11: true,
      },
      {
        name: 'verify',
        description: 'Ejecuta todos los tests, verificaciones de tipos y análisis de seguridad automáticamente. Sin pasos manuales.',
        v11: true,
      },
      {
        name: 'clean',
        description: 'Elimina código muerto y simplifica lo construido — déjalo más limpio de como lo encontraste.',
        v11: false,
      },
      {
        name: 'archive',
        description: 'Guarda todo — código, specs, decisiones — con trazabilidad completa para referencia futura.',
        v11: false,
      },
    ],
    parallel: 'spec + design se ejecutan en paralelo',
    v11Badge: 'Mejorado en v1.1',
  },
  semiFormal: {
    title: 'Razonamiento Semi-Formal',
    subtitle: 'v1.1 — Pensamiento estructurado en fases críticas',
    protocols: [
      {
        name: 'Exploración Estructurada',
        phase: 'explore',
        description:
          'Antes de leer cualquier archivo, declara lo que esperas encontrar. Después, confirma o corrige la predicción. Fuerza investigación genuina, no suposiciones.',
        steps: [
          'Declara lo que esperas antes de ver el archivo',
          'Anota lo que realmente encontraste — con referencias de archivo y línea',
          'Actualiza tu comprensión: confirmada, refutada o refinada',
          'Explica por qué el siguiente archivo es el paso lógico',
        ],
      },
      {
        name: 'Lectura Estructurada',
        phase: 'apply',
        description:
          'Antes de modificar un archivo, declara qué patrones usa. Asegura que el nuevo código encaje naturalmente con lo que ya existe.',
        steps: [
          'Declara qué patrones usa probablemente este archivo',
          'Identifica lo que realmente observas después de leerlo',
          'Comprende cómo los patrones existentes limitan tu implementación',
          'Escribe código que siga el estilo establecido en el archivo',
        ],
      },
      {
        name: 'Certificado Semi-Formal',
        phase: 'review',
        description:
          'Obliga al revisor a trazar cada función y buscar activamente formas en que el código podría fallar — no solo confirmar que funciona.',
        steps: [
          'Mapea cada función: qué recibe, qué devuelve, qué hace',
          'Traza los datos desde su creación hasta su consumo final',
          'Busca activamente escenarios de fallo — asume que el código está mal hasta demostrar lo contrario',
        ],
      },
      {
        name: 'Localización de Fallos',
        phase: 'verify',
        description:
          'Cuando fallan los tests, produce un diagnóstico preciso — no solo "se rompió", sino exactamente dónde y por qué, con niveles de confianza.',
        steps: [
          'Describe paso a paso lo que el test espera que suceda',
          'Identifica exactamente dónde el código se desvía de esa expectativa',
          'Asigna un nivel de confianza a cada hallazgo',
        ],
      },
    ],
  },
  contracts: {
    title: 'Contratos de Seguridad',
    subtitle: 'v1.1 — Cada paso tiene su checklist. Sin atajos.',
    description:
      'Como el checklist pre-vuelo de un piloto: cada fase debe demostrar que está lista antes de empezar, y confirmar que su trabajo está completo antes de pasar a la siguiente. Si falta algo, el workflow se detiene — no el desarrollador.',
    phases: [
      {
        name: 'Investigar',
        pre: [
          'La configuración del proyecto está lista',
          'La tarea a investigar está claramente definida',
        ],
        post: [
          'Los hallazgos están documentados',
          'Los archivos relevantes están identificados',
        ],
      },
      {
        name: 'Construir',
        pre: [
          'Existe un plan de diseño',
          'Las tareas están desglosadas y listadas',
        ],
        post: [
          'Las tareas están marcadas como completadas',
          'El código compila sin errores',
        ],
      },
      {
        name: 'Entregar',
        pre: [
          'Todos los controles de calidad han pasado',
          'La revisión de código no tiene problemas bloqueantes',
        ],
        post: [
          'El trabajo está archivado con trazabilidad completa',
          'La documentación está actualizada',
        ],
      },
    ],
  },
  advancedV11: {
    title: 'Funcionalidades Avanzadas v1.1',
    subtitle: 'Optimizaciones respaldadas por investigación para el pipeline SDD',
    eet: {
      title: 'Parada Inteligente',
      description:
        'Si la IA ha intentado y fallado corregir el mismo tipo de error 3 o más veces en sesiones anteriores, deja de intentarlo y escala en vez de perder tiempo en un callejón sin salida conocido.',
      steps: [
        'Identifica el tipo y categoría del error',
        'Busca en memoria el mismo error en sesiones anteriores',
        '3 o más fallos previos → detener y escalar al desarrollador',
        'Sin fallos previos → seguir intentando (hasta 5 intentos)',
      ],
    },
    rubric: {
      title: 'Checklist de Revisión Personalizado',
      description:
        'Antes de revisar cualquier código, la IA genera un checklist adaptado a ese cambio específico — basado en sus requisitos, decisiones de diseño y reglas de calidad.',
      rows: [
        {
          criterion: 'Requisito satisfecho',
          source: 'Documento de spec',
          weight: 'CRITICAL',
        },
        {
          criterion: 'Reglas de calidad seguidas',
          source: 'Libro de reglas',
          weight: 'CRITICAL',
        },
        {
          criterion: 'Arquitectura respetada',
          source: 'Plan de diseño',
          weight: 'REQUIRED',
        },
      ],
    },
  },
  comparison: {
    title: 'v1.0 → v1.1',
    subtitle: 'Qué cambió y por qué',
    before: {
      title: 'v1.0' as const,
      items: [
        'Exploración superficial — archivos leídos sin propósito ni hipótesis',
        'Aprobaciones automáticas — "se ve bien" sin trazar las funciones',
        'Ciclos de corrección ciegos — 5 intentos sin importar la experiencia previa',
        'Sin contratos — las fases podían lanzarse sin las entradas requeridas',
        'Criterios de evaluación genéricos — basados en buenas prácticas, no en el spec',
        'Reportes de fallo vagos — "el test falló" sin diagnóstico de la causa raíz',
      ],
    },
    after: {
      title: 'v1.1' as const,
      items: [
        'Exploración guiada por hipótesis — cada lectura tiene un propósito declarado y nivel de confianza',
        'Revisión adversarial — trazado de funciones, análisis de flujo de datos, verificación de contra-hipótesis',
        'Parada inteligente — terminación anticipada basada en Engram para patrones sin salida conocidos',
        'Contratos PARCER — pre/post-condiciones formales validadas por el orquestador',
        'Rúbricas dinámicas — criterios generados a partir de specs, diseño y AGENTS.md',
        'Localización de fallos — diagnóstico preciso con referencias de archivo y línea',
      ],
    },
  },
  subAgent: {
    title: 'Estrategia de Sub-Agentes',
    subtitle: 'El modelo correcto para el trabajo correcto',
    rows: [
      {
        phase: 'explore, propose, spec, tasks',
        model: 'Sonnet',
        reason:
          'Salida basada en plantillas con formatos estructurados. Reconocimiento de patrones, no razonamiento profundo.',
      },
      {
        phase: 'design',
        model: 'Opus',
        reason:
          'Decisiones de arquitectura que definen toda la implementación. Los compromisos requieren razonamiento contextual profundo.',
      },
      {
        phase: 'apply',
        model: 'Opus',
        reason:
          'Escribe código de producción. Debe seguir las convenciones del lenguaje, respetar patrones existentes y manejar casos límite.',
      },
      {
        phase: 'review, verify, clean, archive',
        model: 'Sonnet',
        reason:
          'Comparación contra checklists, ejecución de comandos, detección de código muerto y operaciones de archivos.',
      },
    ],
    costNote: '~60–70% de reducción de costos frente a todo-Opus',
  },
  quality: {
    title: 'Aseguramiento de Calidad',
    subtitle: 'Seguimiento continuo de calidad a lo largo del pipeline',
    timelineTitle: 'Seguimiento de Calidad Automático',
    timelineDescription:
      'Después de cada fase, se registra automáticamente un snapshot de calidad. El proceso se documenta a sí mismo — sin reportes manuales.',
    fields: [
      {
        name: 'agentStatus',
        description: '¿Esta fase tuvo éxito o falló?',
      },
      {
        name: 'issues.critical',
        description: 'Número de problemas bloqueantes encontrados',
      },
      {
        name: 'buildHealth',
        description: '¿Los tests, verificaciones de tipos y linting están pasando?',
      },
      {
        name: 'completeness',
        description: '¿Cuánto está hecho? Tareas completadas y requisitos cubiertos.',
      },
      {
        name: 'scope',
        description: '¿Qué cambió? Archivos creados y modificados.',
      },
    ],
    analyticsTitle: 'Panel de Calidad',
    analyticsList: [
      'Progresión de salud del build en todas las fases',
      'Dónde se introducen los problemas (por fase)',
      'Curvas de completitud a lo largo del tiempo',
      'Estimaciones de duración por fase',
      'Detección automática de regresiones',
    ],
  },
  whenToUse: {
    title: '¿Cuándo usar SDD?',
    subtitle: 'La estructura tiene costo — no todo cambio necesita 11 fases',
    spectrum: [
      {
        level: 'Trivial',
        description: 'Typos, actualizaciones de versión, cambios de configuración',
        approach: 'Editar el archivo directamente — sin SDD',
      },
      {
        level: 'Pequeño',
        description: 'Añadir un campo, nueva ruta siguiendo un patrón existente',
        approach: '/sdd:explore + edición manual',
      },
      {
        level: 'Mediano',
        description: 'Funcionalidades que tocan 3–10 archivos con requisitos claros',
        approach: '/sdd:ff + /sdd:apply + /sdd:verify',
      },
      {
        level: 'Grande',
        description: 'Cambios transversales, múltiples dominios, 10+ archivos',
        approach: 'Pipeline completo de 11 fases',
      },
      {
        level: 'Arquitectura',
        description: 'Nuevos módulos, cambio de flujo de datos, cambios sensibles a seguridad',
        approach: 'Pipeline completo con ciclos de revisión adicionales',
      },
    ],
  },
  caseStudy: {
    title: 'Construido con Shenron',
    projectName: 'Gravity Room',
    projectDesc: 'Tracker de entrenamiento de fuerza en producción — monorepo full-stack TypeScript',
    stats: [
      { value: '147.822', label: 'Líneas de Código' },
      { value: '294', label: 'Commits' },
      { value: '13', label: 'Días' },
      { value: '422', label: 'Archivos TypeScript' },
      { value: '67', label: 'Archivos de Tests' },
      { value: '95/100', label: 'Puntuación de Calidad' },
      { value: '62', label: 'Componentes React' },
      { value: '29', label: 'Migraciones de BD' },
      { value: '0', label: 'Regresiones de Build' },
    ],
    techStack: ['React 19', 'ElysiaJS', 'PostgreSQL', 'Drizzle ORM', 'Zod 4', 'TanStack Query', 'Tailwind 4', 'Bun', 'Playwright'],
    sddNote: '141 artefactos de especificación en 14 cambios rastreados',
  },
  cta: {
    title: 'Por qué lo construí',
    motivation:
      'Me cansé de ver a la IA cometer los mismos errores una y otra vez. Cada sesión nueva olvidaba lo que la anterior había aprendido. Cada revisión aprobaba sin cuestionar código que el propio modelo había escrito. Quería un sistema donde la IA no pudiera tomar atajos — donde cada decisión quedara documentada, cada revisión fuera independiente y cada sesión retomara donde la anterior terminó.',
    closing: 'Shenron es el resultado.',
    discordLabel: 'Háblame por Discord',
    discordTag: 'raisen1340',
    copied: '¡Copiado!',
    designedBy: 'Diseñado por',
    author: 'RecheDev',
  },
};
