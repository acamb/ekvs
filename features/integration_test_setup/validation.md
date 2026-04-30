# Validation: integration_test_setup

## Checklist

### Images build without errors

```bash
make integration-test
# oppure
make integration-test-passphrase
```

Expected: tutti gli stage completano, nessun errore di build.

---

### Three containers are running

```bash
docker compose -f tests/integration/docker-compose.nopass.yml ps
```

Expected: `server`, `cli`, `tui` tutti in stato `running`.

---

### Server is reachable from cli container

```bash
docker exec -it integration-cli-1 sh -c "wget -qO- http://server:8080/health || echo 'no health endpoint'"
```

Expected: risposta HTTP dal server (o messaggio di connessione riuscita).

---

### Nopass scenario — CLI can authenticate and list projects

```bash
docker exec -it integration-cli-1 /cli projects list
```

Expected: lista vuota `[]`, nessun errore di autenticazione.

---

### Nopass scenario — full CRUD via CLI

```bash
docker exec -it integration-cli-1 sh
# dentro il container:
/cli projects create myproject
/cli secrets set myproject MY_KEY "hello world"
/cli export myproject
```

Expected: `MY_KEY=hello world` stampato su stdout (valore decriptato).

---

### Nopass scenario — TUI launches and shows project list

```bash
docker exec -it integration-tui-1 /tui
```

Expected:
- La TUI si renderizza nel terminale.
- Il profilo `integration` è pre-caricato.
- La lista progetti è vuota (o riflette i dati creati via CLI nello stesso scenario).

---

### Passphrase scenario — TUI prompts for passphrase

```bash
make integration-test-passphrase
docker exec -it integration-tui-1 /tui
```

Expected: la TUI chiede la passphrase SSH; inserendo `changeme` l'autenticazione va a buon fine.

---

### Private key absent from server image

```bash
docker run --rm $(docker compose -f tests/integration/docker-compose.nopass.yml config --images | grep server) \
  sh -c "ls /root/.ssh 2>/dev/null || echo 'no ssh dir'"
```

Expected: `no ssh dir`.

---

### Storage is empty on fresh start

```bash
docker exec integration-server-1 sh -c "ls /data"
```

Expected: solo la directory `.keys/`, nessun file di progetto.
