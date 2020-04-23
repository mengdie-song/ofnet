package ofctrl

import (
	"encoding/binary"
	"math/rand"
	"net"

	"github.com/contiv/libOpenflow/openflow13"
	"github.com/contiv/libOpenflow/protocol"
	"github.com/contiv/libOpenflow/util"
)

type PacketOut struct {
	InPort  uint32
	OutPort uint32
	Actions []OFAction

	SrcMAC     net.HardwareAddr
	DstMAC     net.HardwareAddr
	IPHeader   *protocol.IPv4
	TCPHeader  *protocol.TCP
	UDPHeader  *protocol.UDP
	ICMPHeader *protocol.ICMP

	ARPHeader *protocol.ARP
}

func (p *PacketOut) GetMessage() util.Message {
	packetOut := openflow13.NewPacketOut()
	packetOut.InPort = p.InPort
	for _, act := range p.Actions {
		packetOut.AddAction(act.GetActionMessage())
	}
	packetOut.Data = p.getEthernetHeader()
	if p.OutPort > 0 {
		packetOut.AddAction(openflow13.NewActionOutput(p.OutPort))
	} else {
		packetOut.AddAction(openflow13.NewActionOutput(openflow13.P_TABLE))
	}
	return packetOut
}

func (p *PacketOut) getEthernetHeader() *protocol.Ethernet {
	ethPkt := &protocol.Ethernet{
		HWDst: p.DstMAC,
		HWSrc: p.SrcMAC,
	}

	var data util.Message
	var ethType uint16
	if p.ARPHeader != nil {
		data = p.ARPHeader
		ethType = 0x0806
	} else {
		switch {
		case p.TCPHeader != nil:
			p.IPHeader.Protocol = protocol.Type_TCP
			p.IPHeader.Data = p.TCPHeader
		case p.UDPHeader != nil:
			p.IPHeader.Protocol = protocol.Type_UDP
			p.IPHeader.Data = p.UDPHeader
		case p.ICMPHeader != nil:
			p.IPHeader.Protocol = protocol.Type_ICMP
			p.IPHeader.Data = p.ICMPHeader
		default:
			p.IPHeader.Protocol = 0xff
		}
		data = p.IPHeader
		ethType = 0x0800
	}
	ethPkt.Ethertype = ethType
	ethPkt.Data = data
	return ethPkt
}

func (p *PacketIn) GetMatches() *Matchers {
	matches := make([]*MatchField, 0, len(p.Match.Fields))
	for i := range p.Match.Fields {
		matches = append(matches, NewMatchField(&p.Match.Fields[i]))
	}
	return &Matchers{matches: matches}
}

func GenerateTCPPacket(srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP, dstPort, srcPort uint16, tcpFlags *uint8) *PacketOut {
	tcpHeader := GenerateTCPHeader(dstPort, srcPort, tcpFlags)
	ipHeader := &protocol.IPv4{
		Version:        4,
		IHL:            5,
		Length:         20 + tcpHeader.Len(),
		Id:             uint16(rand.Int()),
		Flags:          0,
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       protocol.Type_TCP,
		Checksum:       0,
		NWSrc:          srcIP,
		NWDst:          dstIP,
	}
	pktOut := &PacketOut{
		SrcMAC:    srcMAC,
		DstMAC:    dstMAC,
		IPHeader:  ipHeader,
		TCPHeader: tcpHeader,
	}

	return pktOut
}

func GenerateSimpleIPPacket(srcMAC, dstMAC net.HardwareAddr, srcIP, dstIP net.IP) *PacketOut {
	icmpHeader := GenerateICMPHeader(nil, nil)
	ipHeader := &protocol.IPv4{
		Version:        4,
		IHL:            5,
		Length:         20 + icmpHeader.Len(),
		Id:             uint16(rand.Int()),
		Flags:          0,
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       protocol.Type_ICMP,
		Checksum:       0,
		NWSrc:          srcIP,
		NWDst:          dstIP,
	}
	pktOut := &PacketOut{
		SrcMAC:     srcMAC,
		DstMAC:     dstMAC,
		IPHeader:   ipHeader,
		ICMPHeader: icmpHeader,
	}
	return pktOut
}

func GenerateTCPHeader(dstPort, srcPort uint16, flags *uint8) *protocol.TCP {
	header := protocol.NewTCP()
	if dstPort != 0 {
		header.PortDst = dstPort
	} else {
		header.PortDst = uint16(rand.Uint32())
	}
	if srcPort != 0 {
		header.PortSrc = srcPort
	} else {
		header.PortSrc = uint16(rand.Uint32())
	}
	header.AckNum = rand.Uint32()
	header.AckNum = header.AckNum + 1
	header.HdrLen = 20
	if flags != nil {
		header.Code = *flags
	} else {
		header.Code = uint8(1 << 1)
	}
	return header
}

func GenerateICMPHeader(icmpType, icmpCode *uint8) *protocol.ICMP {
	header := protocol.NewICMP()
	if icmpType != nil {
		header.Type = *icmpType
	} else {
		header.Type = 8
	}
	if icmpCode != nil {
		header.Code = *icmpCode
	} else {
		header.Code = 0
	}
	identifier := uint16(rand.Uint32())
	seq := uint16(1)
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data, identifier)
	binary.BigEndian.PutUint16(data[2:], seq)
	return header
}
