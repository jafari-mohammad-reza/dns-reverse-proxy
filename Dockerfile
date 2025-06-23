FROM golang:1.24.1-alpine
ENV GOPROXY=https://goproxy.io,direct
WORKDIR /opt/app
COPY go.* . 
RUN go mod tidy
COPY . .
RUN go build -o dns-proxy
EXPOSE 53
EXPOSE 5353
CMD [ "dns-proxy" ]