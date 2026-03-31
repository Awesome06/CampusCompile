# Frontend Agent Rules

## Tech Stack & Key Libraries
* **Framework**: React 19 + Vite
* **Styling**: Tailwind CSS, PostCSS, Autoprefixer
* **Routing**: `react-router-dom`
* **Network**: `axios`
* **Icons & UI**: `lucide-react`
* **Data Viz**: `recharts`
* **Code Editor**: `@monaco-editor/react` (VS Code engine)
* **Markdown/Math**: `react-markdown`, `remark-gfm`, `remark-math`, `rehype-katex`, `react-latex-next`, `katex`

## Architecture & Conventions
* **Components / Pages**: Use functional components with hooks. Maintain separation of state and presentation.
* **Context/State**: Utilize React Context (`src/context/`) for global state like auth/roles. 
* **Hooks**: Place custom reusable logic in `src/hooks/`.
* **Services**: Isolated API calls go into `src/services/` using `axios`.
* **Telemetry**: Aggressively monitor and flag suspicious behavior (e.g., tab-switching, focus loss, unauthorized copy-pasting, auto-typer checks) to maintain academic integrity. Avoid disabling or bypassing these checks.

## Style & Best Practices
* Strive for a "buttery-smooth" Single Page Application experience essential for embedding the heavy Monaco Code Editor.
* Ensure responsive and accessible UI components.
* Maintain clear application routing mappings (`AppRoutes.jsx`), primarily reflecting distinct views for Admin/Faculty (The Forge) vs Students (The Arena).

## Developer Workflow
* **Start Dev Server**: `npm run dev`
* **Linting**: `npm run lint`
* **Build**: `npm run build`
