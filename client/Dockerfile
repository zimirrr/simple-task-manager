## STAGE 1 : Build the client
FROM simpletaskmanager/stm-client-base:1.0.0 as builder

COPY ./ /stm-client/
WORKDIR /stm-client/

RUN NODE_OPTIONS="--max_old_space_size=4096" npm run build-prod

## STAGE 2 : Configure nginx and run client
FROM nginx:1.18.0

COPY nginx.conf /etc/nginx/conf.d/default.conf

RUN rm -rf /usr/share/nginx/html/*
COPY --from=builder ./stm-client/dist/simple-task-manager /usr/share/nginx/html

CMD [ "nginx", "-g", "daemon off;" ]
