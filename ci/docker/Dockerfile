FROM ubuntu:22.04

RUN apt-get update && apt-get install -y nginx

COPY bin/nginx-hosts-reload-amd64 /usr/local/bin/nginx-hosts-reload
RUN chmod +x /usr/local/bin/nginx-hosts-reload

COPY ci/docker/nginx.conf /etc/nginx/nginx.conf
COPY ci/docker/index.html /var/www/html/index.html

EXPOSE 80

CMD ["sh", "-c", "echo '127.0.0.1 testserver' >> /etc/hosts && nginx -g 'daemon off;'"]
