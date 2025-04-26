# TemperatureByCep
O que este programa faz:

Roda dois webservers o serverA recebe um CEP e valida se o CEP possui 8 digitos, caso tenha ele envia uma chamada POST para a api do serverB. O Server B recebe o numero de CEP válido e retorna o nome da cidade do CEP com as temperaturas da previsão do tempo para o dia atual na cidade solicitada.

Os sistemas utilizam OTEL + Zipkin, com tracing distribuído entre Serviço A - Serviço B.
Também está sendo utilizado span para medir o tempo de resposta do serviço de busca de CEP e busca de temperatura# OtelAndZipkinOnServices
