FROM node:20-alpine

WORKDIR /app
COPY vendor/lib-stupidmailjavascript-0.1.0.tgz /vendor/lib-stupidmailjavascript-0.1.0.tgz
COPY mailer/package.json mailer/package-lock.json ./
RUN npm ci --omit=dev
COPY mailer/src ./src
COPY mailer/templates ./templates
RUN mkdir -p /data && chown -R node:node /app /data

ENV MAILER_PORT=8080 \
    MAILER_DATA_DIR=/data \
    MAILER_TEMPLATE_DIR=/app/templates

USER node
EXPOSE 8080
CMD ["node", "src/server.js"]
