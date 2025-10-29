<script lang="ts">
  import { onMount } from 'svelte';
  let progress = 0;
  let status = 'Aguardando início';
  let processing = false;

  async function startJob() {
    processing = true;
    status = 'Solicitando trabalho...';

    try {
      const res = await fetch('http://localhost:8080/job', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ white: true })
      });
      const job = await res.json();

      status = 'Processando...';
      const wasm = await import('../pkg/wasm.js');
      const result = await wasm.process(new Uint8Array(job.payload));

      // Simula progresso visual enquanto processa
      const interval = setInterval(() => {
        if (progress < 100) progress += 1;
        else clearInterval(interval);
      }, 100);

      await fetch('http://localhost:8080/result', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jobId: job.jobId, result: Array.from(result) })
      });

      status = 'Concluído com sucesso';
    } catch (err) {
      console.error(err);
      status = 'Erro durante o processamento';
    } finally {
      processing = false;
    }
  }
</script>

<main class="container">
  <h1>Distribuição de Trabalho WASM</h1>

  <div class="status">
    <p>{status}</p>
    {#if processing}
      <p>Progresso: {progress}%</p>
      <div class="bar">
        <div class="fill" style="width: {progress}%"></div>
      </div>
    {/if}
  </div>

  <button on:click={startJob} disabled={processing}>Iniciar Trabalho</button>
</main>

<style>
  .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    font-family: system-ui, sans-serif;
    background: #101820;
    color: #fff;
  }
  button {
    margin-top: 2rem;
    background: #00bfa6;
    color: #fff;
    border: none;
    padding: 1rem 2rem;
    border-radius: 10px;
    cursor: pointer;
    font-size: 1.1rem;
    transition: background 0.3s;
  }
  button:disabled {
    background: #555;
    cursor: not-allowed;
  }
  button:hover:not(:disabled) {
    background: #00a38a;
  }
  .bar {
    width: 300px;
    height: 10px;
    background: #333;
    border-radius: 5px;
    overflow: hidden;
    margin-top: 0.5rem;
  }
  .fill {
    height: 100%;
    background: #00bfa6;
    transition: width 0.1s linear;
  }
</style>