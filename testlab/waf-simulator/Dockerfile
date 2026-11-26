FROM nginx:alpine
RUN apk add --no-cache openssl iproute2 && \
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/nginx/server.key -out /etc/nginx/server.crt \
    -subj "/CN=localhost"
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80 443
STOPSIGNAL SIGQUIT
CMD ["nginx", "-g", "daemon off;"]
