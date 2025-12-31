<script>
  import { onMount } from 'svelte';
  import { Network } from 'vis-network';

  export let ideas = [];
  export let relations = [];

  let container;
  let network;

  // COLORES NEÓN PARA EL MODO OSCURO
  const colors = {
    apoya: '#4ade80',      // Green 400
    contradice: '#f87171', // Red 400
    refina: '#60a5fa',     // Blue 400
    depende_de: '#94a3b8'  // Slate 400
  };

  $: nodosVis = ideas.map(idea => ({
    id: idea.ID,
    label: idea.Text, 
    shape: 'dot', 
    value: 10 + (idea.score_apoyo || 0) * 3, 
    color: {
      // Lógica de Semáforo Neón
      background: 
        idea.estado_colectivo === 'casi_consensuada' ? '#22c55e' : // Verde Éxito
        idea.estado_colectivo === 'controvertida' ? '#ef4444' :    // Rojo Peligro
        idea.estado_colectivo === 'en_riesgo' ? '#f97316' :        // Naranja
        idea.estado_colectivo === 'en_debate' ? '#eab308' :        // Amarillo
        '#64748b', // Gris (Emergente)
      border: '#ffffff', // Borde blanco para resaltar en fondo oscuro
      highlight: { background: '#8b5cf6', border: '#fff' } // Púrpura al seleccionar
    },
    font: { 
      multi: true, 
      size: 16, 
      face: 'monospace', // Toque Hacker
      color: '#f8fafc',  // Texto Blanco
      strokeWidth: 2,    // Borde negro al texto para leerlo mejor
      strokeColor: '#0f172a'
    }
  }));

  $: aristasVis = relations.map(rel => ({
    from: rel.FromID,
    to: rel.ToID,
    arrows: 'to',
    color: { color: colors[rel.Type] || '#ccc', opacity: 0.8 },
    dashes: rel.Type === 'depende_de',
    width: 2
  }));

  onMount(() => {
    const data = { nodes: nodosVis, edges: aristasVis };
    const options = {
      physics: {
        enabled: true,
        barnesHut: {
          gravitationalConstant: -3000,
          centralGravity: 0.3,
          springLength: 200,
        },
        minVelocity: 0.75
      },
      nodes: { borderWidth: 2, shadow: true },
      edges: { smooth: { type: 'dynamic' } },
      
      // CONFIGURACIÓN VISUAL DEL NETWORK
      interaction: { hover: true, tooltipDelay: 200 },
      layout: { randomSeed: 2 }, // Para que no salten tanto al recargar
    };

    network = new Network(container, data, options);
  });

  $: if (network && (ideas || relations)) {
    network.setData({ nodes: nodosVis, edges: aristasVis });
  }
</script>

<div class="lienzo-container">
  <div class="lienzo-cerebro" bind:this={container}></div>
  <div class="watermark">CMP v0.2 // VISUALIZER</div>
</div>

<style>
  .lienzo-container {
    position: relative;
    width: 100%;
    margin-bottom: 30px;
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid #334155; /* Borde sutil */
    box-shadow: 0 0 20px rgba(139, 92, 246, 0.1); /* Resplandor púrpura muy suave */
  }

  .lienzo-cerebro {
    width: 100%;
    height: 500px; /* Más inmersivo */
    background: #0f172a; /* Fondo Dark Slate */
    background-image: radial-gradient(#1e293b 1px, transparent 1px);
    background-size: 20px 20px; /* Efecto grilla sutil */
  }

  .watermark {
    position: absolute;
    bottom: 10px;
    right: 15px;
    font-family: 'Courier New', monospace;
    font-size: 0.7rem;
    color: #475569;
    pointer-events: none;
  }
</style>