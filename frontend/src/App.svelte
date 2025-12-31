<script>
  import { onMount } from 'svelte';
  import Cerebro from './lib/Cerebro.svelte'; 
  
  // -- ESTADO USUARIO --
  let usuarioActual = "";
  let rolUsuario = "proponente"; 
  let codigoEquipo = ""; 
  let nombreEquipoReal = "";
  
  // -- LOGIN INPUTS --
  let modoLogin = "unirse";
  let inputNombre = "";
  let inputRol = "proponente"; 
  let inputCodigoUnirse = "";
  let inputCodigoCrear = "";
  let inputNombreEquipo = "";

  // -- APP DATA --
  let ideas = [];      
  let relations = [];
  let mostrarFinalizadas = false;
  let mostrarGuia = true; 

  // -- FORMULARIOS --
  let textIdea = "";
  let typeIdea = "propuesta"; 
  let idOrigen = "";
  let idDestino = "";
  let tipoRelacion = "apoya"; 

  onMount(async () => {
    const u = localStorage.getItem('cmp_user');
    const r = localStorage.getItem('cmp_role');
    const t = localStorage.getItem('cmp_team');
    const n = localStorage.getItem('cmp_team_name');
    if (u && t) {
      usuarioActual = u; rolUsuario = r || 'participante'; codigoEquipo = t; nombreEquipoReal = n || t;
      await cargarDatos();
    }
  });

  const unirseEquipo = async () => {
    if (!inputNombre.trim() || !inputCodigoUnirse.trim()) return alert("ERROR: Faltan credenciales.");
    const codigo = inputCodigoUnirse.trim().toLowerCase();
    const res = await fetch(`http://localhost:8080/check-team?id=${codigo}`);
    if (res.ok) {
      const data = await res.json();
      iniciarSesion(inputNombre.trim(), inputRol, data.id, data.name);
    } else { alert("ERROR 404: Equipo no encontrado en la red."); }
  }

  const crearEquipo = async () => {
    if (!inputNombre.trim() || !inputCodigoCrear.trim()) return alert("ERROR: Faltan parámetros.");
    const codigo = inputCodigoCrear.trim().toLowerCase().replace(/\s+/g, '-');
    const res = await fetch('http://localhost:8080/create-team', {
      method: 'POST', body: JSON.stringify({ id: codigo, name: inputNombreEquipo })
    });
    if (res.status === 201) {
      iniciarSesion(inputNombre.trim(), inputRol, codigo, inputNombreEquipo);
    } else { alert("ERROR 409: Código de frecuencia ocupado."); }
  }

  const iniciarSesion = (u, r, c, n) => {
    usuarioActual = u; rolUsuario = r; codigoEquipo = c; nombreEquipoReal = n;
    localStorage.setItem('cmp_user', u); 
    localStorage.setItem('cmp_role', r);
    localStorage.setItem('cmp_team', c); 
    localStorage.setItem('cmp_team_name', n);
    cargarDatos();
  }

  const salir = () => { if(confirm("¿Desconectar del servidor?")) { localStorage.clear(); usuarioActual = ""; ideas = []; } }

  const cargarDatos = async () => {
    try {
      const [resIdeas, resRel] = await Promise.all([
        fetch(`http://localhost:8080/?team=${codigoEquipo}`),
        fetch(`http://localhost:8080/connect?team=${codigoEquipo}`)
      ]);
      if (resIdeas.ok) { 
        ideas = await resIdeas.json() || []; 
        relations = await resRel.json() || []; 
      }
    } catch (e) { console.error("Error de Conexión:", e); }
  };

  $: ideasVisibles = ideas.filter(i => mostrarFinalizadas || i.estado_colectivo !== 'finalizada');

  const sendIdea = async () => {
    if (!textIdea.trim()) return;
    await fetch('http://localhost:8080', { 
      method: 'POST', 
      body: JSON.stringify({ 
        ID: Date.now().toString(), 
        TeamID: codigoEquipo, 
        Autor_Id: usuarioActual, 
        RolAutor: rolUsuario, 
        Text: textIdea, 
        Type: typeIdea 
      }) 
    });
    textIdea = ""; await cargarDatos();
  }

  const conectarIdeas = async () => {
    if (idOrigen === idDestino) return alert("Error lógico: Auto-conexión.");
    await fetch('http://localhost:8080/connect', { 
      method: 'POST', 
      body: JSON.stringify({ TeamID: codigoEquipo, FromID: idOrigen, ToID: idDestino, Type: tipoRelacion }) 
    });
    await cargarDatos();
  }

  const votar = async (ideaID, tipo, intensidad) => {
    await fetch('http://localhost:8080/interact', { 
      method: 'POST', 
      body: JSON.stringify({ IdeaID: ideaID, UserID: usuarioActual, Type: tipo, Intensidad: intensidad }) 
    });
    await cargarDatos();
  }

  const finalizarIdea = async (id) => {
    if(confirm("¿Archivar decisión consensuada?")) {
      await fetch('http://localhost:8080/resolve', { method: 'POST', body: JSON.stringify({ IdeaID: id }) });
      await cargarDatos();
    }
  }
</script>

{#if !usuarioActual}
  <div class="login-screen">
    <div class="terminal-window">
      <div class="terminal-header">
        <span class="dot red"></span><span class="dot yellow"></span><span class="dot green"></span>
        <span class="title">bash — protocolo_v0.2</span>
      </div>
      
      <div class="terminal-body">
        <div class="boot-text">
          <p>> Inicializando Interfaz CMP...</p>
          <p>> Conexión Segura... <span class="ok">ESTABLECIDA</span></p>
        </div>

        <div class="ascii-tabs">
          <button class:active={modoLogin==='unirse'} on:click={()=>modoLogin='unirse'}>[ UNIRSE_EQUIPO ]</button>
          <button class:active={modoLogin==='crear'} on:click={()=>modoLogin='crear'}>[ CREAR_NUEVO ]</button>
        </div>

        <div class="input-zone">
          <div class="line">
            <span class="prompt">usuario@identidad:~$</span>
            <input type="text" placeholder="identificate_aqui" bind:value={inputNombre} />
          </div>
          
          <div class="line">
            <span class="prompt">usuario@rol:~$</span>
            <select bind:value={inputRol} class="terminal-select">
              <option value="proponente">PROPONENTE</option>
              <option value="critico">CRITICO</option>
              <option value="sintetizador">SINTETIZADOR</option>
            </select>
          </div>

          {#if modoLogin === 'unirse'}
            <div class="line">
              <span class="prompt">red@conectar:~$</span>
              <input type="text" placeholder="codigo_equipo_destino" bind:value={inputCodigoUnirse} on:keydown={(e) => e.key === 'Enter' && unirseEquipo()} />
            </div>
            <button class="btn-terminal" on:click={unirseEquipo}>./EJECUTAR_UNION.SH</button>
          {:else}
            <div class="line">
              <span class="prompt">sis@nuevo_nombre:~$</span>
              <input type="text" placeholder="nombre_visual_equipo" bind:value={inputNombreEquipo} />
            </div>
            <div class="line">
              <span class="prompt">sis@id_unico:~$</span>
              <input type="text" placeholder="codigo_acceso_equipo" bind:value={inputCodigoCrear} />
            </div>
            <button class="btn-terminal create" on:click={crearEquipo}>./MKDIR_EQUIPO.SH</button>
          {/if}
        </div>
      </div>
    </div>
  </div>
{:else}
  <main>
    <header>
      <div class="brand">
        <h1>FFL <span class="highlight">CMP</span></h1>
        <small>{nombreEquipoReal} <span style="color:#64748b">({codigoEquipo})</span></small>
      </div>
      <div class="controls">
        <span class="badge">👤 {usuarioActual} <span class="role">[{rolUsuario}]</span></span>
        <button class="btn-logout" on:click={salir}>CERRAR</button>
        <label class="toggle"><input type="checkbox" bind:checked={mostrarFinalizadas}> <span>Historial</span></label>
      </div>
    </header>

    <div class="guide-panel">
      <div class="guide-header" on:click={() => mostrarGuia = !mostrarGuia}>ℹ️ Leyenda del Sistema {mostrarGuia ? '▼' : '▶'}</div>
      {#if mostrarGuia}
        <div class="guide-content">
          <span>🟢 Consenso</span> <span>🔴 Controvertida</span> <span>🟠 En Riesgo</span>
          <span class="sep">|</span>
          <span>👍 Apoyo(+1)</span> <span>🔥 Convicción(+3)</span> <span>🛡️ Duda(-1)</span> <span>🛑 Bloqueo(-3)</span>
        </div>
      {/if}
    </div>

    <Cerebro ideas={ideasVisibles} {relations} />

    <div class="action-grid">
      <div class="panel glass">
        <h3>Flujo de Entrada</h3>
        <input type="text" placeholder="Ingresar datos..." bind:value={textIdea} />
        <select bind:value={typeIdea}>
          <option value="propuesta">Propuesta</option>
          <option value="pregunta">Pregunta</option>
          <option value="preocupacion">Preocupación</option>
          <option value="dato">Dato</option>
        </select>
        <button class="btn-action" on:click={sendIdea}>ENVIAR PAQUETE</button>
      </div>

      <div class="panel glass">
        <h3>Enlazador Sináptico</h3>
        <div class="row">
          <select bind:value={idOrigen}><option value="" disabled selected>Origen</option>{#each ideasVisibles as i}<option value={i.ID}>{i.Text.substring(0,15)}...</option>{/each}</select>
          <select bind:value={tipoRelacion} class="rel-{tipoRelacion}">
            <option value="apoya">🟢 Apoya</option>
            <option value="contradice">🔴 Contradice</option>
            <option value="refina">🔵 Refina</option>
            <option value="depende_de">⚪ Depende</option>
          </select>
          <select bind:value={idDestino}><option value="" disabled selected>Destino</option>{#each ideasVisibles as i}{#if i.ID!==idOrigen}<option value={i.ID}>{i.Text.substring(0,15)}...</option>{/if}{/each}</select>
        </div>
        <button class="btn-action" on:click={conectarIdeas}>ENLAZAR</button>
      </div>
    </div>

    <div class="cards">
      {#each ideasVisibles as idea}
        <div class="card {idea.estado_colectivo}">
          <div class="top">
            <span class="tag">{idea.Type}</span> 
            <span class="status-dot {idea.estado_colectivo}"></span>
          </div>
          <p class="text">{idea.Text}</p>
          <div class="meta">
            <small>{idea.Autor_id} :: {idea.RolAutor}</small> 
            <small class="status-txt">{idea.estado_colectivo}</small>
          </div>
          
          <div class="votes">
            <div class="group support">
              <button on:click={()=>votar(idea.ID,'apoyo',1)} title="Ok (+1)">👍</button>
              <button class="fire" on:click={()=>votar(idea.ID,'apoyo',3)} title="Apoyo Total (+3)">🔥</button>
              <span class="val">{idea.score_apoyo || 0}</span>
            </div>
            <div class="group object">
              <span class="val">{idea.score_objecion || 0}</span>
              <button on:click={()=>votar(idea.ID,'objecion',1)} title="Duda (1)">🤔</button>
              <button class="stop" on:click={()=>votar(idea.ID,'objecion',3)} title="Bloqueo (3)">🛑</button>
            </div>
          </div>

          {#if idea.estado_colectivo === 'casi_consensuada'}
            <button class="btn-resolve" on:click={()=>finalizarIdea(idea.ID)}>✅ EJECUTAR</button>
          {/if}
        </div>
      {/each}
    </div>
  </main>
{/if}

<style>
  :global(body) { background: #050505; color: #0f0; font-family: 'Courier New', monospace; margin: 0; }
  main { max-width: 1000px; margin: 0 auto; padding: 20px; }
  
  /* TERMINAL LOGIN */
  .login-screen { height: 100vh; display: flex; justify-content: center; align-items: center; background: #000; }
  .terminal-window { width: 500px; background: #0c0c0c; border: 1px solid #333; box-shadow: 0 0 20px rgba(0, 255, 0, 0.2); }
  .terminal-header { background: #1a1a1a; padding: 5px 10px; display: flex; gap: 6px; border-bottom: 1px solid #333; }
  .dot { width: 10px; height: 10px; border-radius: 50%; }
  .red { background: #ff5f56; } .yellow { background: #ffbd2e; } .green { background: #27c93f; }
  .title { margin: 0 auto; color: #666; font-size: 0.8rem; }
  
  .terminal-body { padding: 20px; color: #0f0; }
  .boot-text p { margin: 2px 0; font-size: 0.8rem; color: #0a0; }
  .ok { color: #0f0; float: right; }

  .ascii-tabs { display: flex; margin: 20px 0; border-bottom: 1px solid #333; }
  .ascii-tabs button { flex: 1; background: transparent; border: none; color: #444; cursor: pointer; font-family: inherit; font-weight: bold; padding: 10px; }
  .ascii-tabs button.active { color: #0f0; background: #111; border: 1px solid #333; border-bottom: none; }
  
  .input-zone { margin-top: 20px; }
  .line { display: flex; align-items: center; margin-bottom: 10px; border-bottom: 1px solid #111; }
  .prompt { color: #0a0; margin-right: 10px; font-weight: bold; }
  input, .terminal-select { background: transparent; border: none; color: #fff; width: 100%; outline: none; font-family: inherit; font-size: 1rem; }
  .terminal-select { cursor: pointer; }
  .terminal-select option { background: #000; color: #0f0; }
  
  .btn-terminal { width: 100%; margin-top: 20px; background: #0f0; color: #000; border: none; padding: 10px; font-weight: bold; cursor: pointer; font-family: inherit; }
  .btn-terminal:hover { background: #0a0; }
  .btn-terminal.create { background: #8a2be2; color: #fff; } /* Violeta Hacker para crear */

  /* APP UI */
  header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; border-bottom: 1px solid #333; padding-bottom: 10px; }
  .brand h1 { margin: 0; } .highlight { color: #0f0; }
  .controls { display: flex; gap: 10px; align-items: center; }
  .badge { border: 1px solid #333; padding: 5px; font-size: 0.8rem; }
  .role { color: #888; }
  .btn-logout { background: transparent; border: 1px solid #f00; color: #f00; cursor: pointer; font-family: inherit; }
  
  .cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 15px; }
  .card { background: #111; padding: 15px; border: 1px solid #333; display: flex; flex-direction: column; }
  .card:hover { border-color: #0f0; }
  
  /* Bordes Semánticos */
  .card.casi_consensuada { border-top: 3px solid #0f0; }
  .card.controvertida { border-top: 3px solid #f00; }
  .card.emergente { border-top: 3px solid #444; }
  
  .top { display: flex; justify-content: space-between; margin-bottom: 10px; }
  .tag { background: #222; padding: 2px 6px; font-size: 0.7rem; color: #aaa; }
  .status-dot { width: 8px; height: 8px; border-radius: 50%; }
  .casi_consensuada .status-dot { background: #0f0; box-shadow: 0 0 5px #0f0; }
  .controvertida .status-dot { background: #f00; box-shadow: 0 0 5px #f00; }
  
  .text { color: #eee; margin: 0 0 15px 0; font-size: 1rem; }
  .meta { display: flex; justify-content: space-between; color: #666; font-size: 0.7rem; margin-bottom: 10px; }
  
  /* Voting */
  .votes { display: flex; justify-content: space-between; background: #080808; padding: 5px; border: 1px solid #222; }
  .group { display: flex; align-items: center; gap: 5px; }
  .votes button { background: #111; border: 1px solid #333; color: #666; width: 25px; height: 25px; cursor: pointer; display: flex; justify-content: center; align-items: center; }
  .votes button:hover { color: #fff; border-color: #fff; }
  .val { font-weight: bold; color: #aaa; margin: 0 5px; }
  .support .val { color: #0f0; } .object .val { color: #f00; }
  
  .btn-resolve { margin-top: 10px; width: 100%; background: #0f0; color: #000; border: none; padding: 8px; cursor: pointer; font-weight: bold; }
  
  /* Panels */
  .action-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 20px; }
  .panel { padding: 15px; border: 1px solid #333; background: #0a0a0a; display: flex; flex-direction: column; gap: 10px; }
  .panel h3 { margin: 0; color: #0a0; font-size: 0.9rem; text-transform: uppercase; }
  input, select { background: #000; border: 1px solid #333; color: #fff; padding: 8px; }
  .btn-action { background: #222; color: #fff; border: 1px solid #444; padding: 8px; cursor: pointer; }
  .btn-action:hover { border-color: #0f0; color: #0f0; }
  .row { display: flex; gap: 5px; } .row select { flex: 1; }
  
  .guide-panel { margin-bottom: 20px; background: #111; border: 1px solid #333; padding: 10px; font-size: 0.8rem; color: #888; cursor: pointer; }
  .guide-content { margin-top: 10px; display: flex; gap: 15px; color: #aaa; }
</style>