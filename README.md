# http-multicaster

## Description
Le service fonctionne comme un proxy minimaliste.
Il accèpte des requêtes HTTP et les reproduit sur un ou plusieurs backends.

## Configuration
Le service se configure via des variables d'environnement:

- BACKENDS: liste des backends (au format IP:PORT) séparés par des ',' sur lesquels diffuser la requête d'origine. 

    Exemple:

        BACKENDS=10.0.1.10:3128,10.0.1.11:3129

- LISTEN: addresse d'écoute du service au format IP:PORT

    Exemple:

        LISTEN=10.0.1.80:8080

- HTTP_CLIENT_TIMEOUT: timeout en millisecondes pour les appels HTTP vers les backends.

    Exemple:

        HTTP_CLIENT_TIMEOUT=500
        
Un exemple de fichier de définition du service est disponible dans les sources.

Le service effectue les requêtes vers les backends en parallèle et génère un rapport en réponse à la requête d'origine.

## Compilation

    CGO_ENABLE=0 go build
