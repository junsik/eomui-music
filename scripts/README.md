# scripts — 보조 스크립트

빌드에 꼭 필요하지는 않고, 아이콘을 다시 만들 때만 씁니다.

## generate_icons.py

빨간 음표 아이콘을 만들어 확장과 앱에 넣습니다.

```bash
pip install pillow
python scripts/generate_icons.py
```

만드는 파일:

| 경로 | 용도 |
|---|---|
| `extension/icons/icon{16,48,128}.png` | 크롬 확장 아이콘 |
| `app/icon.ico` | 트레이 아이콘 (`//go:embed`로 exe에 포함) |

`app/icon.ico`를 바꾸면 앱을 다시 빌드해야 반영됩니다.
