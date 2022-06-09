FROM golang:alpine

RUN apk add --no-cache git

WORKDIR /data
RUN git clone https://github.com/sopov/resumeio2pdf.git /data/app
WORKDIR /data/app
RUN go build

COPY entrypoint.sh /data/app/entrypoint.sh

ENTRYPOINT [ "/data/app/entrypoint.sh" ]