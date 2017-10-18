FROM alpine:3.6
MAINTAINER  Florian Forster <ff@octo.it>

RUN apk add --no-cache ca-certificates clang
COPY ./clang-format-gae /app/clang-format-gae

ENTRYPOINT ["/app/clang-format-gae"]
EXPOSE 8080
