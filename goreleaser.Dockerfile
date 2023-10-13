FROM scratch

COPY cob /bin/cob

ENTRYPOINT ["/bin/cob"]