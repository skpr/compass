FROM skpr/nginx-drupal:dev-v2-latest

# Copy the Drupal application code in.
COPY --from=drupal:10.2 --chown=skpr:skpr /opt/drupal /data

# Make this Drupal application compatible with our images.
RUN ln -sf /data/web /data/app
