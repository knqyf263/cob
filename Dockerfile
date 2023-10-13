FROM golang:1.21 AS binary

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 go build -o /bin/cob -ldflags "-s -w -extldflags '-static'" .

FROM scratch

COPY --from=binary /bin/cob /bin/cob

ENTRYPOINT ["/bin/cob"]