# Create Table: {{ TableName }}
# ------------------------------------------------------------

CREATE TABLE `{{ TableName }}` ({% for field in Fields %}
  {{ field }}{% endfor %}
) ENGINE={{ TableEngine }}{% if AutoIncrement %} AUTO_INCREMENT={{ autoIncrement }}{% endif %} DEFAULT CHARSET={{ TableCharset }};
