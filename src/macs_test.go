package main

import (
	"net"
	"testing"
	"testing/quick"
)

func TestMAC1(t *testing.T) {
	dev1 := randDevice(t)
	dev2 := randDevice(t)

	defer dev1.Close()
	defer dev2.Close()

	peer1, _ := dev2.NewPeer(dev1.privateKey.publicKey())
	peer2, _ := dev1.NewPeer(dev2.privateKey.publicKey())

	assertEqual(t, peer1.mac.keyMAC1[:], dev1.mac.keyMAC1[:])
	assertEqual(t, peer2.mac.keyMAC1[:], dev2.mac.keyMAC1[:])

	msg1 := make([]byte, 256)
	copy(msg1, []byte("some content"))
	peer1.mac.AddMacs(msg1)
	if dev1.mac.CheckMAC1(msg1) == false {
		t.Fatal("failed to verify mac1")
	}
}

func TestMACs(t *testing.T) {
	assertion := func(
		addr net.UDPAddr,
		addrInvalid net.UDPAddr,
		sk1 NoisePrivateKey,
		sk2 NoisePrivateKey,
		msg []byte,
		receiver uint32,
	) bool {
		device1 := randDevice(t)
		device1.SetPrivateKey(sk1)

		device2 := randDevice(t)
		device2.SetPrivateKey(sk2)

		defer device1.Close()
		defer device2.Close()

		peer1, _ := device2.NewPeer(device1.privateKey.publicKey())
		peer2, _ := device1.NewPeer(device2.privateKey.publicKey())

		if addr.Port < 0 {
			return true
		}

		addr.Port &= 0xffff

		if len(msg) < 32 {
			return true
		}

		assertEqual(t, peer1.mac.keyMAC1[:], device1.mac.keyMAC1[:])
		assertEqual(t, peer2.mac.keyMAC1[:], device2.mac.keyMAC1[:])

		device2.indices.Insert(receiver, IndexTableEntry{
			peer:      peer1,
			handshake: &peer1.handshake,
		})

		// test just MAC1

		peer1.mac.AddMacs(msg)
		if device1.mac.CheckMAC1(msg) == false {
			return false
		}

		// exchange cookie reply

		cr, err := device1.CreateMessageCookieReply(msg, receiver, &addr)
		if err != nil {
			return false
		}

		if !device2.ConsumeMessageCookieReply(cr) {
			return false
		}

		// test MAC1 + MAC2

		peer1.mac.AddMacs(msg)
		if !device1.mac.CheckMAC1(msg) {
			return false
		}
		if !device1.mac.CheckMAC2(msg, &addr) {
			return false
		}

		// test invalid

		if device1.mac.CheckMAC2(msg, &addrInvalid) {
			return false
		}
		msg[5] ^= 1
		if device1.mac.CheckMAC1(msg) {
			return false
		}

		t.Log("Passed")

		return true
	}

	err := quick.Check(assertion, nil)
	if err != nil {
		t.Error(err)
	}
}
