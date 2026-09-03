import struct

def make_whatsapp_icon():
    # 32x32 RGBA icon
    width = 32
    height = 32
    # BMP / DIB header & pixels (bottom-up)
    pixels = []
    # WhatsApp green: #25D366 (RGBA: 37, 211, 102, 255)
    # Background transparent: (0,0,0,0)
    center_x = 15.5
    center_y = 15.5
    radius = 14.0
    for y in range(height):
        row = []
        for x in range(width):
            dist = ((x - center_x)**2 + (y - center_y)**2)**0.5
            if dist <= radius:
                # Inner phone shape or green circle
                # simple circle with white inner dot
                inner_dist = ((x - 16)**2 + (y - 16)**2)**0.5
                if 4.0 <= inner_dist <= 8.0 and (x >= 14 or y >= 14):
                    # White icon feature
                    row.extend([255, 255, 255, 255])
                else:
                    # Green bubble
                    row.extend([37, 211, 102, 255])
            else:
                row.extend([0, 0, 0, 0])
        pixels.append(bytes(row))

    xor_mask = b''.join(pixels)
    and_mask = b'\x00' * (width * height // 8) # 1-bit mask

    # ICONDIR (6 bytes)
    # idReserved=0, idType=1, idCount=1
    icondir = struct.pack('<HHH', 0, 1, 1)

    # ICONDIRENTRY (16 bytes)
    # bWidth, bHeight, bColorCount, bReserved, wPlanes, wBitCount, dwBytesInRes, dwImageOffset
    image_size = 40 + len(xor_mask) + len(and_mask)
    offset = 6 + 16
    direntry = struct.pack('<BBBBHHII', width, height, 0, 0, 1, 32, image_size, offset)

    # BITMAPINFOHEADER (40 bytes)
    # biSize, biWidth, biHeight (x2 for ICO), biPlanes, biBitCount, biCompression, biSizeImage, biXPelsPerMeter, biYPelsPerMeter, biClrUsed, biClrImportant
    bih = struct.pack('<IIIHHIIIIII', 40, width, height * 2, 1, 32, 0, len(xor_mask), 0, 0, 0, 0)

    with open('icon.ico', 'wb') as f:
        f.write(icondir + direntry + bih + xor_mask + and_mask)

make_whatsapp_icon()
