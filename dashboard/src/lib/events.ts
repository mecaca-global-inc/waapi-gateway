export type WebhookEvent = {
  id: string;
  label: string;
  description: string;
  group: string;
};

export const WEBHOOK_EVENTS: WebhookEvent[] = [
  { id: "message", label: "Incoming message", description: "A new message was received in any chat.", group: "Messages" },
  { id: "message.ack", label: "Message ack / receipt", description: "Delivery, read, or played status update.", group: "Messages" },
  { id: "message.reaction", label: "Message reaction", description: "Someone reacted with an emoji.", group: "Messages" },
  { id: "message.revoked", label: "Message revoked", description: "A message was deleted for everyone.", group: "Messages" },
  { id: "message.edited", label: "Message edited", description: "A message was edited after sending.", group: "Messages" },

  { id: "session.status", label: "Session status change", description: "STARTING / SCAN_QR / WORKING / FAILED / STOPPED.", group: "Session" },
  { id: "state.qr", label: "QR code refreshed", description: "A new QR string is available.", group: "Session" },
  { id: "state.pair", label: "Pair success", description: "Device successfully paired.", group: "Session" },
  { id: "state.loggedout", label: "Logged out", description: "WhatsApp logged the device out.", group: "Session" },

  { id: "group.joined", label: "Group joined", description: "Account joined a new group.", group: "Groups" },
  { id: "group.participants", label: "Group participants change", description: "Members added, removed, promoted, demoted.", group: "Groups" },

  { id: "call.offer", label: "Incoming call", description: "Someone is calling.", group: "Calls" },
];

export function findEvent(id: string): WebhookEvent | undefined {
  return WEBHOOK_EVENTS.find((e) => e.id === id);
}
