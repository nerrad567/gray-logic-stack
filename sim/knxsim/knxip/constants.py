# KNXnet/IP protocol constants
# Reference: KNX Standard 03_08_02 (Core), 03_08_04 (Tunnelling)

# Header
HEADER_SIZE = 0x06
PROTOCOL_VERSION = 0x10

# Service type identifiers (2 bytes, big-endian)
CONNECT_REQUEST = 0x0205
CONNECT_RESPONSE = 0x0206
CONNECTION_STATE_REQUEST = 0x0207
CONNECTION_STATE_RESPONSE = 0x0208
DISCONNECT_REQUEST = 0x0209
DISCONNECT_RESPONSE = 0x020A
TUNNELLING_REQUEST = 0x0420
TUNNELLING_ACK = 0x0421

# Connection types (CRI - Connection Request Information)
TUNNEL_CONNECTION = 0x04

# Tunnelling layer
TUNNEL_LINKLAYER = 0x02

# Error codes
E_NO_ERROR = 0x00
E_CONNECTION_TYPE = 0x22
E_CONNECTION_OPTION = 0x23
E_NO_MORE_CONNECTIONS = 0x24
E_DATA_CONNECTION = 0x26
E_KNX_CONNECTION = 0x27

# Host Protocol Address Information (HPAI)
HPAI_SIZE = 0x08
IPV4_UDP = 0x01

# cEMI message codes
L_DATA_REQ = 0x11  # From client to knxd (request to send)
L_DATA_CON = 0x2E  # Confirmation from bus
L_DATA_IND = 0x29  # Indication from bus (device → client)

# cEMI control fields
CTRL1_STANDARD = 0xBC  # Standard frame, no repeat, broadcast, normal priority
CTRL2_GROUP_HOP6 = 0xE0  # Group address destination, hop count 6

# APCI (Application Protocol Control Information) - high nibble of first data byte
APCI_GROUP_READ = 0x00
APCI_GROUP_RESPONSE = 0x40
APCI_GROUP_WRITE = 0x80
