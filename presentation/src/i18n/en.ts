export const en = {
  nav: { prev: 'Previous', next: 'Next' },
  hero: {
    hookLine: '147,822 lines of TypeScript. 13 days. One developer.',
    hookSub: 'This is what happens when you stop prompting and start orchestrating.',
    title: 'SHENRON',
    subtitle: 'Language-Agnostic Spec-Driven Development',
    tagline: 'Gather the specs. Summon the code.',
    version: 'v1.1',
    versionLabel: 'Any Stack \u00b7 Semi-Formal Reasoning',
  },
  problem: {
    title: 'The Problem',
    subtitle: 'Six failure modes of standard AI-assisted coding',
    items: [
      {
        title: 'Context Pollution',
        description: 'One agent reads the whole codebase. Attention dilutes — wrong files get edited.',
      },
      {
        title: 'Session Amnesia',
        description: 'Every session starts from zero. Decisions, bugs fixed, conventions — all forgotten.',
      },
      {
        title: 'No Traceability',
        description: 'Code ships with no record of why. Months later, no one knows.',
      },
      {
        title: 'Hallucinations',
        description: 'Full context forces compression. Constraints disappear; invented logic appears.',
      },
      {
        title: 'No Verification',
        description: 'The AI reviews its own code — same author, same blind spots.',
      },
      {
        title: 'Stack Lock-in',
        description: 'Tooling hardcodes one language. Switch stacks and you start from scratch.',
      },
    ],
  },
  insight: {
    quote: "You don't need a smarter model.",
    quoteLine2: 'You need a smarter workflow.',
    description:
      'The same model produces dramatically better output when given a clean context with focused instructions and the right reference material. The problem is not model intelligence \u2014 it is workflow architecture: how we structure the interaction between the human, the model, and the codebase.',
  },
  architecture: {
    title: 'Architecture',
    subtitle: 'How the pieces fit together',
    orchestrator: {
      title: 'Orchestrator',
      description:
        'The project coordinator. Assigns tasks, tracks progress, and asks for your approval at every step. Never writes code itself—it delegates everything.',
    },
    subAgents: {
      title: 'Specialist Workers',
      description:
        'A fresh AI assistant for each phase, each with only the information it needs. No distractions, no information overload.',
    },
    artifacts: {
      title: 'Living Documentation',
      description:
        'Every decision, plan, and review stored as files. Nothing is ever lost or misremembered—the project documents itself.',
    },
    memory: {
      title: 'Semantic Memory',
      description:
        'A self-hosted RAG server that remembers decisions, solved bugs, and patterns across all sessions. Searches by meaning, not keywords—finds what you need even when you use different words.',
    },
    skills: {
      title: 'Expertise Guides',
      description:
        'Up-to-date knowledge files for each technology. Ensures the AI uses React 19 patterns for React 19—never outdated answers.',
    },
  },
  pillars: {
    title: 'The Five Pillars',
    subtitle: 'Each pillar solves a specific failure mode of standard AI coding',
    items: [
      {
        title: 'Divide & Specialize',
        description:
          'Each step is handled by a fresh AI with only what it needs. Like hiring a specialist for each task instead of one overloaded generalist.',
        detail:
          'The implementation agent never sees your entire codebase—only the design plan and the specific file it is modifying. Focused context produces focused results.',
      },
      {
        title: 'Semantic Memory',
        description:
          'A RAG-powered memory that persists decisions, bugs, and patterns across sessions. Hybrid vector + keyword search finds relevant context even when vocabulary differs.',
        detail:
          'Self-hosted on your infrastructure (Qdrant + Ollama). Searches by meaning: "auth expiry" finds "JWT token renewal" automatically. No manual keyword enrichment needed.',
      },
      {
        title: 'Code Review Rulebook',
        description:
          'A written set of rules that an independent reviewer checks against every change. The reviewer never wrote the code it reviews—no bias, no blind spots.',
        detail:
          'Rules use clear keywords: REJECT (hard block), REQUIRE (must justify), PREFER (advisory). Every rule is versioned and transparent.',
      },
      {
        title: 'Up-to-Date Expertise',
        description:
          'Self-updating knowledge guides for each technology. The AI knows which version of each tool you are using and applies the right patterns.',
        detail:
          'If a guide does not have the answer, it searches the internet and updates itself. Knowledge gaps close over time.',
      },
      {
        title: 'Semi-Formal Reasoning',
        description:
          'Forces the AI to reason through problems step by step before acting. Like making a surgeon write a plan before the first cut.',
        detail:
          'Four structured thinking protocols applied at the most critical phases: investigation, implementation, review, and verification.',
        isV11: true,
      },
    ],
  },
  pipeline: {
    title: 'The Pipeline',
    subtitle: '11 phases \u00b7 any language, any stack',
    phases: [
      {
        name: 'init',
        description:
          'Auto-detect your tech stack and generate project configuration.',
        v11: false,
      },
      {
        name: 'explore',
        description:
          'Investigate the codebase—understand what exists before changing anything.',
        v11: true,
      },
      {
        name: 'propose',
        description:
          'Write a plain-English proposal: what changes, why, and how to roll back if needed.',
        v11: false,
      },
      {
        name: 'spec',
        description:
          'Define formal requirements: exactly what the code must do, with acceptance criteria.',
        v11: false,
      },
      {
        name: 'design',
        description:
          'Plan the technical architecture before writing a single line of code.',
        v11: false,
      },
      {
        name: 'tasks',
        description:
          'Break the design into a dependency-ordered implementation checklist.',
        v11: false,
      },
      {
        name: 'apply',
        description:
          'Write the code following bottom-up task order, with automatic build verification after each batch.',
        v11: true,
      },
      {
        name: 'review',
        description:
          'Independent code review against requirements, rules, and the original design.',
        v11: true,
      },
      {
        name: 'verify',
        description:
          'Run all tests, type checks, and security scans automatically. No manual steps.',
        v11: true,
      },
      {
        name: 'clean',
        description:
          'Remove dead code and simplify what was built—leave it cleaner than you found it.',
        v11: false,
      },
      {
        name: 'archive',
        description:
          'Save everything—code, specs, decisions—with full traceability for future reference.',
        v11: false,
      },
    ],
    parallel: 'spec + design run in parallel',
    v11Badge: 'Enhanced in v1.1',
  },
  semiFormal: {
    title: 'Semi-Formal Reasoning',
    subtitle: 'v1.1 \u2014 Structured thinking at critical phases',
    protocols: [
      {
        name: 'Structured Exploration',
        phase: 'explore',
        description:
          'Before reading any file, declare what you expect to find. After reading, confirm or correct the prediction. Forces genuine investigation, not assumption.',
        steps: [
          'State what you expect before looking at the file',
          'Note what you actually found—with file and line references',
          'Update your understanding: confirmed, refuted, or refined',
          'Explain why the next file is the logical next step',
        ],
      },
      {
        name: 'Structured Reading',
        phase: 'apply',
        description:
          'Before modifying a file, declare what patterns it uses. Ensures new code fits naturally with what already exists.',
        steps: [
          'Declare what patterns this file likely uses',
          'Identify what you actually observe after reading',
          'Understand how existing patterns constrain your implementation',
          "Write code that follows the file's established style",
        ],
      },
      {
        name: 'Semi-Formal Certificate',
        phase: 'review',
        description:
          'Forces the reviewer to trace every function and actively search for ways the code could fail—not just confirm it works.',
        steps: [
          'Map every function: what it receives, what it returns, what it does',
          'Trace data from creation to final consumption',
          'Actively search for failure scenarios—assume the code is wrong until proven otherwise',
        ],
      },
      {
        name: 'Fault Localization',
        phase: 'verify',
        description:
          'When tests fail, produce a precise diagnosis—not just "it broke", but exactly where and why, with confidence levels.',
        steps: [
          'Describe step by step what the test expects to happen',
          'Pinpoint exactly where the code diverges from that expectation',
          'Assign a confidence level to each finding',
        ],
      },
    ],
  },
  contracts: {
    title: 'Safety Contracts',
    subtitle: 'v1.1 — Every step has a checklist. No shortcuts.',
    description:
      'Like a pilot\'s pre-flight checklist: each phase must prove it\'s ready before starting, and confirm its work is complete before handing off. If something is missing, the workflow stops — not the developer.',
    phases: [
      {
        name: 'Investigate',
        pre: [
          'Project configuration is ready',
          'The task to investigate is clearly defined',
        ],
        post: [
          'Findings are documented',
          'Relevant files are identified',
        ],
      },
      {
        name: 'Build',
        pre: [
          'A design plan exists',
          'Tasks are broken down and listed',
        ],
        post: [
          'Tasks are marked as completed',
          'Code compiles without errors',
        ],
      },
      {
        name: 'Ship',
        pre: [
          'All quality checks have passed',
          'Code review has no blocking issues',
        ],
        post: [
          'Work is archived with full traceability',
          'Documentation is updated',
        ],
      },
    ],
  },
  advancedV11: {
    title: 'Advanced v1.1 Features',
    subtitle: 'Research-backed optimizations for the SDD pipeline',
    eet: {
      title: 'Smart Early Stopping',
      description:
        'If the AI has tried and failed to fix the same type of error 3+ times in past sessions, it stops trying and escalates instead of wasting time on a known dead end.',
      steps: [
        'Identify the type and category of the error',
        'Semantic search for the same error pattern in past sessions',
        '3 or more prior failures found → stop and escalate to the developer',
        'No prior failures → keep trying (up to 5 attempts)',
      ],
    },
    rubric: {
      title: 'Custom Review Checklist',
      description:
        'Before reviewing any code, the AI generates a checklist tailored to that specific change—based on its requirements, design decisions, and quality rules.',
      rows: [
        {
          criterion: 'Requirement satisfied',
          source: 'Spec document',
          weight: 'CRITICAL',
        },
        {
          criterion: 'Quality rules followed',
          source: 'Rulebook',
          weight: 'CRITICAL',
        },
        {
          criterion: 'Architecture respected',
          source: 'Design plan',
          weight: 'REQUIRED',
        },
      ],
    },
  },
  comparison: {
    title: 'v1.0 \u2192 v1.1',
    subtitle: 'What changed and why',
    before: {
      title: 'v1.0' as const,
      items: [
        'Shallow exploration \u2014 files read without purpose or hypothesis',
        'Rubber-stamp reviews \u2014 "looks good" without function tracing',
        'Blind fix loops \u2014 5 attempts regardless of prior experience',
        'No contracts \u2014 phases could launch without required inputs',
        'Generic evaluation criteria \u2014 anchored to best practices, not the spec',
        'Vague failure reports \u2014 "test failed" without root cause diagnosis',
      ],
    },
    after: {
      title: 'v1.1' as const,
      items: [
        'Hypothesis-driven exploration \u2014 every file read has a declared purpose and confidence',
        'Adversarial review \u2014 function tracing, data flow analysis, counter-hypothesis check',
        'Smart EET \u2014 memory-backed early termination using semantic search for known dead-end patterns',
        'PARCER contracts \u2014 formal pre/post-conditions validated by the orchestrator',
        'Dynamic agentic rubrics \u2014 criteria generated from specs, design, and project conventions',
        'Fault localization \u2014 PREMISES + DIVERGENCE CLAIMS with File:Line references',
      ],
    },
  },
  subAgent: {
    title: 'Sub-Agent Strategy',
    subtitle: 'Right model for the right job',
    rows: [
      {
        phase: 'explore, propose, spec, tasks',
        model: 'Sonnet',
        reason:
          'Template-driven output with structured formats. Pattern matching, not deep reasoning.',
      },
      {
        phase: 'design',
        model: 'Opus',
        reason:
          'Architecture decisions that shape the entire implementation. Trade-offs require deep contextual reasoning.',
      },
      {
        phase: 'apply',
        model: 'Opus',
        reason:
          'Writes production code. Must follow language conventions, match existing patterns, and handle edge cases.',
      },
      {
        phase: 'review, verify, clean, archive',
        model: 'Sonnet',
        reason:
          'Checklist comparison, command execution, pattern matching for dead code, and file operations.',
      },
    ],
    costNote: '~60\u201370% cost reduction vs. all-Opus',
  },
  quality: {
    title: 'Quality Assurance',
    subtitle: 'Continuous quality tracking across the pipeline',
    timelineTitle: 'Automatic Quality Tracking',
    timelineDescription:
      'After every phase, a quality snapshot is recorded automatically. The process documents itself—no manual reporting needed.',
    fields: [
      {
        name: 'agentStatus',
        description: 'Did this phase succeed or fail?',
      },
      {
        name: 'issues.critical',
        description: 'Number of blocking issues found',
      },
      {
        name: 'buildHealth',
        description: 'Are tests, type checks, and linting passing?',
      },
      {
        name: 'completeness',
        description: 'How much is done? Tasks completed and requirements covered.',
      },
      {
        name: 'scope',
        description: 'What changed? Files created and modified.',
      },
    ],
    analyticsTitle: 'Quality Dashboard',
    analyticsList: [
      'Build health progression across all phases',
      'Where issues are introduced (by phase)',
      'Completion curves over time',
      'Phase duration estimates',
      'Automatic regression detection',
    ],
  },
  whenToUse: {
    title: 'When to Use SDD',
    subtitle: 'Works with any language \u2014 structure scales to the change',
    spectrum: [
      {
        level: 'Trivial',
        description: 'Typos, version bumps, config value changes',
        approach: 'Just edit the file \u2014 no SDD',
      },
      {
        level: 'Small',
        description: 'Single field addition, new route following existing pattern',
        approach: '/sdd:explore + manual edit',
      },
      {
        level: 'Medium',
        description: 'Features touching 3\u201310 files with clear requirements',
        approach: '/sdd:ff + /sdd:apply + /sdd:verify',
      },
      {
        level: 'Large',
        description:
          'Cross-cutting concerns, multiple domains, 10+ files',
        approach: 'Full 11-phase pipeline',
      },
      {
        level: 'Architecture',
        description:
          'New modules, changed data flow, security-sensitive changes',
        approach: 'Full pipeline with extra review cycles',
      },
    ],
  },
  caseStudy: {
    title: 'Built with Shenron',
    projectName: 'Gravity Room',
    projectDesc: 'Production strength training tracker — full-stack TypeScript monorepo',
    stats: [
      { value: '147,822', label: 'Lines of Code' },
      { value: '294', label: 'Commits' },
      { value: '13', label: 'Days' },
      { value: '422', label: 'TypeScript Files' },
      { value: '67', label: 'Test Files' },
      { value: '95/100', label: 'Quality Score' },
      { value: '62', label: 'React Components' },
      { value: '29', label: 'DB Migrations' },
      { value: '0', label: 'Build Regressions' },
    ],
    techStack: ['React 19', 'ElysiaJS', 'PostgreSQL', 'Drizzle ORM', 'Zod 4', 'TanStack Query', 'Tailwind 4', 'Bun', 'Playwright'],
    sddNote: '141 specification artifacts across 14 tracked changes',
  },
  cta: {
    title: 'Why I Built This',
    motivation:
      'I got tired of watching AI make the same mistakes over and over. Every new session forgot what the last one learned. Every review rubber-stamped code the model itself wrote. I wanted a system where the AI couldn\'t cut corners — where every decision was documented, every review was independent, and every session picked up where the last one left off.',
    closing: 'Shenron is the result.',
    discordLabel: 'Talk to me on Discord',
    discordTag: 'raisen1340' as const,
    copied: 'Copied!',
    designedBy: 'Designed by',
    author: 'RecheDev' as const,
  },
} as const;

type DeepString<T> = T extends string
  ? string
  : T extends readonly (infer U)[]
    ? readonly DeepString<U>[]
    : T extends object
      ? { readonly [K in keyof T]: DeepString<T[K]> }
      : T;

export type Translations = DeepString<typeof en>;
