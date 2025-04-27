<template>
  <div class="wrap">
    <header class="hero">
      <img src="./assets/logo.png" class="logo" alt="native planet logo" />
      <div class="title">
        <h1>Urbit for Umbrel</h1>
        <h4>
          <img src="https://nyc3.digitaloceanspaces.com/startram/np-tiny.png" class="inline-logo" alt="np logo" />
          by <a href="https://nativeplanet.io">Native Planet Corp</a>
        </h4>
      </div>
    </header>

    <main>
      <router-view />
    </main>
  </div>
</template>

<script setup>
</script>

<style scoped>
.wrap {
  max-width: 48rem;
  margin: 1.5rem auto 0;
  padding: 0 10px;
}
.hero {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 1rem 10px;
  border: 2px solid var(--accent);
  border-radius: 6px;
  background: var(--card-bg);
}
.logo {
  width: 4rem;
  height: auto;
}
.title {
  flex: 1;
  text-align: center;
}
.title h1 {
  margin: 0;
  font-size: 1.6rem;
}
.title h4 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 400;
  color: var(--accent-lite);
  display: flex;
  align-items: center;
  justify-content: center;
}
.inline-logo {
  height: 1em;
  width: auto;
}

@media (max-width: 480px) {
  .hero {
    flex-direction: column;
  }
}
</style>
