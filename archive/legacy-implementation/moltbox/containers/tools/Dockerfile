FROM docker:28-cli

RUN apk add --no-cache bash git python3 py3-pip py3-virtualenv

WORKDIR /opt/remram-gateway
COPY . /opt/remram-gateway

RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --upgrade pip \
    && /opt/venv/bin/pip install ./moltbox-cli

ENV PATH=/opt/venv/bin:$PATH
ENV MOLTBOX_REPO_ROOT=/opt/remram-gateway
ENV PYTHONUNBUFFERED=1

WORKDIR /opt/remram-gateway/moltbox-cli

EXPOSE 7474

CMD ["moltbox", "tools", "serve"]
