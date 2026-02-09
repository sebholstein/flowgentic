// Funny bot name generator for Issue Overseers

const prefixes = [
  "Beep",
  "Boop",
  "Buzz",
  "Whirr",
  "Clank",
  "Spark",
  "Byte",
  "Pixel",
  "Glitch",
  "Logic",
  "Proto",
  "Cyber",
  "Robo",
  "Mecha",
  "Turbo",
  "Mega",
  "Ultra",
  "Nano",
  "Giga",
  "Circuit",
  "Binary",
  "Vector",
  "Matrix",
  "Quantum",
  "Servo",
  "Crank",
  "Zippy",
  "Blinky",
  "Chatty",
  "Nerdy",
  "Brainy",
  "Smarty",
];

const suffixes = [
  "Bot",
  "Tron",
  "Matic",
  "Zoid",
  "Unit",
  "Core",
  "Prime",
  "Max",
  "3000",
  "9000",
  "XL",
  "Pro",
  "Plus",
  "Lite",
  "Mini",
  "Alpha",
  "Beta",
  "Omega",
  "Zero",
  "One",
  "Rex",
  "Flux",
  "Spark",
  "Chip",
  "Gear",
  "Bolt",
  "Wire",
  "Node",
  "Loop",
  "Stack",
  "Byte",
  "Bit",
];

// Generate a consistent bot name from an ID (deterministic)
export function generateNameFromId(id: string): string {
  let hash = 0;
  for (let i = 0; i < id.length; i++) {
    const char = id.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash;
  }

  const prefixIndex = Math.abs(hash) % prefixes.length;
  const suffixIndex = Math.abs(hash >> 8) % suffixes.length;

  return `${prefixes[prefixIndex]}-${suffixes[suffixIndex]}`;
}

// Generate a random bot name
export function generateRandomName(): string {
  const prefixIndex = Math.floor(Math.random() * prefixes.length);
  const suffixIndex = Math.floor(Math.random() * suffixes.length);

  return `${prefixes[prefixIndex]}-${suffixes[suffixIndex]}`;
}

// Get initials from a bot name (uses first letter of each part)
export function getNameInitials(name: string): string {
  return name
    .split("-")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}
