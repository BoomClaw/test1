FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o /bin/probe main.go

FROM alpine:3.20
COPY --from=build /bin/probe /bin/probe
EXPOSE 8080
CMD ["/bin/probe"]
