# Waveline Music Server
<a href="https://play.google.com/store/apps/details?id=com.waveline.app" target="_blank">
<img src="https://play.google.com/intl/en_us/badges/images/generic/en_badge_web_generic.png" width="200">
</a>


https://waveline.app/
### Installing

```Sh
$ git clone https://github.com/MihkelBaranov/waveline-go.git
$ cd waveline-go
$ cd go run main.go
```


### API
|                |Description                    |
|----------------|-------------------------------|
|`GET /sync`|Build music library|
|`GET /playlists`|Get all playlists|
|`GET /tracks`|Get tracks|
|`GET /art/:id`| Album art |
|`GET /stream/:id`| Stream audio |
|`GET /favourite/:id`| Toggle favourite |
|`GET /favourites`| Get all favourites|