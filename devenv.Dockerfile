FROM golang:1.23

RUN apt-get update && apt-get install vim --yes

WORKDIR /root/workspace

CMD ["bash"]