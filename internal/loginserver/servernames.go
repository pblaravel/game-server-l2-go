package loginserver

// ServerName matches Java ServerNameDAO.
var serverNames = map[int]string{
	1: "Bartz", 2: "Sieghardt", 3: "Kain", 4: "Lionna", 5: "Erica",
	6: "Gustin", 7: "Devianne", 8: "Hindemith", 9: "Teon (EURO)", 10: "Franz (EURO)",
	11: "Luna (EURO)", 12: "Sayha", 13: "Aria", 14: "Phoenix", 15: "Chronos",
	16: "Naia (EURO)", 17: "Elhwynna", 18: "Ellikia", 19: "Shikken", 20: "Scryde",
	21: "Frikios", 22: "Ophylia", 23: "Shakdun", 24: "Tarziph", 25: "Aria",
	26: "Esenn", 27: "Elcardia", 28: "Yiana", 29: "Seresin", 30: "Tarkai",
	31: "Khadia", 32: "Roien", 33: "Kallint (Non-PvP)", 34: "Baium", 35: "Kamael",
	36: "Beleth", 37: "Anakim", 38: "Lilith", 39: "Thifiel", 40: "Lithra",
	41: "Lockirin", 42: "Kakai", 43: "Cadmus", 44: "Athebaldt", 45: "Blackbird",
	46: "Ramsheart", 47: "Esthus", 48: "Vasper", 49: "Lancer", 50: "Ashton",
	51: "Waytrel", 52: "Waltner", 53: "Tahnford", 54: "Hunter", 55: "Dewell",
	56: "Rodemaye", 57: "Ken Rauhel", 58: "Ken Abigail", 59: "Ken Orwen", 60: "Van Holter",
	61: "Desperion", 62: "Einhovant", 63: "Shunaiman", 64: "Faris", 65: "Tor",
	66: "Carneiar", 67: "Dwyllios", 68: "Baium", 69: "Hallate", 70: "Zaken",
	71: "Core",
}

func ServerName(id int) string {
	if n, ok := serverNames[id]; ok {
		return n
	}
	return "Undefined"
}
