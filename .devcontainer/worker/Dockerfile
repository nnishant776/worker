FROM localhost/worker:base

WORKDIR /root

ENV PS1="[\u@\h \W]# "

COPY Makefile Makefile

# Optional. Remove or change to something else based on the requirements
RUN make postcreate
