#docker build -f ./build/package/Dockerfile -t dispatch ../
#create -p 50000:50000 --name dispatchctr dispatch
#network connect brewery_bridge dispatchctr
# docker logs -f dispatchctr
# docker network create --driver bridge brewery_bridge



FROM golang:alpine as build
WORKDIR /app

COPY ./dispatch /src

RUN cd /src/cmd/dispatch && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/dispatch

FROM alpine
WORKDIR /app
COPY --from=build /app/dispatch .

ENV PORT=50000 \
    TLS_BYPASS=TRUE \
    NETWORK=localhost

CMD ./dispatch

#ENTRYPOINT [ "app/ptaasaiq" ]