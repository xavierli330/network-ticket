FROM node:20-alpine AS builder
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN mkdir -p public
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /build/package.json ./
COPY --from=builder /build/node_modules ./node_modules
COPY --from=builder /build/.next ./.next
COPY --from=builder /build/public ./public
COPY --from=builder /build/next.config.js ./

EXPOSE 3000
CMD ["npm", "start"]
