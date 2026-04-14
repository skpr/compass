FROM docker.io/skpr/node:20-v3-stable

USER root

ARG COMPASS_NODE_VERSION=v0.0.7

RUN wget -q -O /tmp/compass-node.apk "https://github.com/nickschuch/compass-node/releases/download/${COMPASS_NODE_VERSION}/compass-node-$(apk --print-arch).apk" && \
    apk add --allow-untrusted /tmp/compass-node.apk && \
    rm /tmp/compass-node.apk

COPY --chown=skpr:skpr package.json /app/package.json
RUN cd /app && npm install

COPY --chown=skpr:skpr src /app/src
COPY --chown=skpr:skpr app.js /app/app.js

USER skpr

ENV NODE_OPTIONS="--import /usr/lib/compass/node/js/index.mjs"

CMD ["node", "/app/app.js"]

