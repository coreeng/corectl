FROM golang:1.22

RUN apt-get update && apt-get install vim --yes

WORKDIR /root/workspace

CMD ["bash"]