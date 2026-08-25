# addr2gpx

Géocode une liste d'adresses françaises via l'[API Géoplateforme IGN](https://geoservices.ign.fr/documentation/services/services-geoplateforme/geocodage) et génère un fichier GPX de waypoints.

Deux modes, même logique (`gpx.go`) :

- **CLI** (`main.go`, build tag `!js`) : lit les adresses sur stdin, écrit le GPX sur stdout.
- **Web / WASM** (`main_wasm.go`, build tag `js && wasm`) : app statique dans `docs/`, hébergeable sur GitHub Pages. Les appels HTTP passent par `fetch` (l'API IGN autorise le CORS).

## Commandes

```
make cli                          # binaire CLI ./addr2gpx
./addr2gpx < adresses.txt > sortie.gpx
make wasm                         # compile docs/main.wasm + copie wasm_exec.js
make serve                        # preview local sur http://localhost:8000
```

## Déploiement GitHub Pages

1. `make wasm`, commit et push (y compris `docs/main.wasm`).
2. Sur GitHub : Settings → Pages → Source « Deploy from a branch », branche `main`, dossier `/docs`.

Format d'entrée : une adresse par ligne, lignes vides et lignes commençant par `#` ignorées.

## Rate limit

L'API impose 50 req/s. Protection à deux niveaux (`gpx.go`) :

- pacing préventif : 50 ms entre chaque requête (~20 req/s max) ;
- retry sur HTTP 429 : jusqu'à 3 nouvelles tentatives, en respectant `Retry-After` si présent, sinon backoff exponentiel 1s/2s/4s (dans le navigateur, `Retry-After` n'est pas exposé par CORS, c'est donc le backoff qui s'applique).

Tests : `go test ./...` (le chemin 429 est couvert via `httptest`).
