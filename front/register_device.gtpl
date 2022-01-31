{{ template "header" . }}
<div>
    <p>デバイス名登録</p>
    <form action="/register_name" method="post">
        <input type="text" name="device_name" value="{{ .Name }}">
        <button type="submit">登録</button>
    </form>
</div>
{{ template "footer" }}